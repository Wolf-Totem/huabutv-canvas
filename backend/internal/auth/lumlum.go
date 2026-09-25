package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/outbound"
)

const lumlumPublicCanvasURL = "https://api.lumlum.tv/api/v1/public_canvas/"

func (s *Service) FetchLumlumPublicCanvas(id string) (*LibTVImportResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, kernel.BadAuthRequest("画布编号为空")
	}
	req, err := http.NewRequest(http.MethodGet, lumlumPublicCanvasURL+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	outbound.ApplyDefaultOutboundHeaders(req)
	client := outbound.OutboundHTTPClient(20 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("公开画布请求失败")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, libTVMaxResponseBytes+1))
	if err != nil {
		return nil, errors.New("读取公开画布失败")
	}
	if len(body) > libTVMaxResponseBytes {
		return nil, errors.New("公开画布响应过大")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("公开画布请求失败（HTTP %d）", resp.StatusCode)
	}
	detail, err := parsePublicCanvasBody(body, id)
	if err != nil {
		return nil, err
	}
	return adaptLibTVDetail(detail)
}

func parsePublicCanvasBody(body []byte, fallbackID string) (*libTVDetail, error) {
	var envelope libTVEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && len(envelope.Data) > 0 {
		if detail, err := decodePublicCanvasData(envelope.Data, fallbackID); err == nil {
			if envelope.Code != 0 {
				msg := strings.TrimSpace(envelope.Msg)
				if msg == "" {
					msg = "公开画布不可导入"
				}
				return nil, kernel.NotFound(msg)
			}
			return detail, nil
		}
	}
	return decodePublicCanvasData(body, fallbackID)
}

func decodePublicCanvasData(raw json.RawMessage, fallbackID string) (*libTVDetail, error) {
	var detail libTVDetail
	if err := json.Unmarshal(raw, &detail); err == nil && (len(detail.NodeList) > 0 || len(detail.ConnectionList) > 0) {
		if strings.TrimSpace(detail.ProjectMeta.UUID) == "" {
			detail.ProjectMeta.UUID = fallbackID
		}
		return &detail, nil
	}
	var wrapped struct {
		ProjectMeta    libTVDetail          `json:"project"`
		Canvas         libTVDetail          `json:"canvas"`
		NodeList       []libTVRawNode       `json:"nodes"`
		ConnectionList []libTVRawConnection `json:"connections"`
		Name           string               `json:"name"`
		UUID           string               `json:"uuid"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, errors.New("公开画布数据格式无效")
	}
	if len(wrapped.Canvas.NodeList) > 0 {
		detail = wrapped.Canvas
	} else if len(wrapped.ProjectMeta.NodeList) > 0 {
		detail = wrapped.ProjectMeta
	} else {
		detail.NodeList = wrapped.NodeList
		detail.ConnectionList = wrapped.ConnectionList
		detail.ProjectMeta.Name = wrapped.Name
		detail.ProjectMeta.UUID = wrapped.UUID
	}
	if len(detail.NodeList) == 0 {
		return nil, kernel.NotFound("公开画布没有制作过程")
	}
	if strings.TrimSpace(detail.ProjectMeta.UUID) == "" {
		detail.ProjectMeta.UUID = fallbackID
	}
	detail.ProjectMeta.Effective.CanRead = true
	return &detail, nil
}
