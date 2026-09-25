# 佳速图片 接口字段

## 协议身份

- 插件 ID：`jiasu-image`。
- Provider ID：`jiasu-image`。
- 能力：`image`。
- 默认 Base URL：`https://ai.jiasuapi.com`。
- 鉴权驱动：`bearer`。
- 创建：`POST /v1/images/create`。
- 查询：`GET /v1/images/tasks/{{taskId}}`。
- 生命周期：异步任务。创建立即返回 `queued` 回执，必须轮询任务直到完成。

## 配置字段

| 字段 | 类型 | 必填 | 含义 |
| --- | --- | --- | --- |
| `apiKey` | secret | 是 | API Key |

## 统一字段映射

| 统一字段 | 类型 | 必填 | 上游映射 | 说明 |
| --- | --- | --- | --- | --- |
| `model` | string | 是 | `model` | 图片模型 ID。 |
| `prompt` | string | 是 | `prompt` | 图片提示词。 |
| `images` | media[] | 否 | `provider image/reference fields` | 参考图或编辑源图，role 由业务层确定。 |
| `imageCount` | integer | 否 | `n/sample_count` | 输出数量。 |
| `aspectRatio` | string | 否 | `ratio/aspect_ratio` | 宽高比，如 1:1 / 16:9。 |
| `resolution` | string | 否 | `resolution` | 分辨率档位，如 1K / 2K / 4K。 |
| `quality` | string | 否 | `quality` | 质量档位。 |
| `providerOptions` | object | 否 | `provider-specific fields` | 插件命名空间内的厂商扩展字段。 |

## 上游请求模板逐字段清单

下表由插件请求模板生成，覆盖 body、query、headers 和 multipart 文件声明中的每个字段。

| 上游位置 | 值或转换表达式 |
| --- | --- |
| `create.method` | `"POST"` |
| `create.path` | `"/v1/images/create"` |
| `create.contentType` | `"application/json"` |
| `create.body.model` | `{"$ref":"request.model"}` |
| `create.body.prompt` | `{"$ref":"request.prompt"}` |
| `create.body.n` | `{"$omitEmpty":{"$if":{"condition":{"$gt":[{"$ref":"request.imageCount"},0]},"then":{"$ref":"request.imageCount"},"else":1}}}` |
| `create.body.size` | `{"$omitEmpty":{"$ref":"request.extra.size"}}` |
| `create.body.ratio` | `{"$omitEmpty":{"$coalesce":[{"$ref":"request.extra.ratio"},{"$ref":"request.aspectRatio"}]}}` |
| `create.body.aspect_ratio` | `{"$omitEmpty":{"$coalesce":[{"$ref":"request.extra.ratio"},{"$ref":"request.aspectRatio"}]}}` |
| `create.body.resolution` | `{"$omitEmpty":{"$coalesce":[{"$ref":"request.extra.resolution"},{"$ref":"request.resolution"}]}}` |
| `create.body.quality` | `{"$omitEmpty":{"$ref":"request.quality"}}` |
| `create.body.references` | `{"$omitEmpty":{"$map":{"from":{"$sortByOrder":{"$ref":"request.images"}},"as":"media","in":{"$ref":"media.value"}}}}` |

## Provider 扩展键

- `extra.size`：像素尺寸，如 `1024x1024`
- `extra.ratio`：宽高比，如 `16:9`
- `extra.resolution`：分辨率档，如 `1K`

佳速图片必须同时传 `size`、`ratio`（及别名 `aspect_ratio`）和 `resolution`。不要把比例写进 `size`，也不要把 `1K` 写进 `size`。

## 响应映射逐字段清单

| 映射位置 | 上游路径或转换表达式 |
| --- | --- |
| `response.status` | `"succeeded"` |
| `response.images` | `{"$map":{"from":{"$ref":"response.data"},"as":"item","in":{"url":{"$omitEmpty":{"$coalesce":[{"$ref":"item.url"},{"$ref":"item.b64_json"}]}}}}}` |
| `response.usage` | `{"$ref":"response.usage"}` |
| `response.errorPaths[0]` | `"error.code"` |
| `response.messagePaths[0]` | `"error.message"` |

## 响应与错误

插件把上游 task/status/text/media/usage 映射为统一结果。临时媒体 URL 标记为 ephemeral，由宿主立即下载持久化。HTTP 错误、业务 code 和 error object 保持失败语义，不包装成成功。

## 兼容边界

佳速图片是异步接口：`POST /v1/images/create` 只返回 `id/task_id` 和 `status=queued`，再用 `GET /v1/images/tasks/{task_id}` 轮询。完成结果在 `result_urls` / `result_url` / `url`。不要把创建回执当成同步出图，也不要改走 OpenAI `/v1/images/generations`。1K 类模型把档位写在 `resolution`（`1K`），画幅写在 `ratio`，像素写在 `size`（如 `1024x1024`）。参考图用 `images`。

<!-- YINGCE_MANIFEST_CONTRACT_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "yingce.plugin/v2",
  "id": "jiasu-image",
  "name": "佳速图片",
  "version": "2.0.0",
  "author": "佳速API / 影策",
  "description": "佳速图片 独立请求协议插件。",
  "documentation": "<当前插件的完整 documentation，由 README.md 与 docs/interface.md 拼接而成；为避免 JSON 递归，此处不重复展开正文。>",
  "permissions": [
    "generation.run",
    "media.read"
  ],
  "configuration": {
    "fields": [
      {
        "name": "apiKey",
        "type": "secret",
        "label": "API Key",
        "required": true
      }
    ]
  },
  "contributes": {
    "providers": [
      {
        "id": "jiasu-image",
        "label": "佳速图片",
        "capabilities": [
          "image"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "baseUrl": "https://ai.jiasuapi.com",
        "requiresPublicMediaUrls": true,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "图片模型 ID。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": true,
            "mapping": "prompt",
            "description": "图片提示词。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "provider image/reference fields",
            "description": "参考图或编辑源图，role 由业务层确定。"
          },
          {
            "name": "imageCount",
            "type": "integer",
            "required": false,
            "mapping": "n/sample_count",
            "description": "输出数量。"
          },
          {
            "name": "aspectRatio",
            "type": "string",
            "required": false,
            "mapping": "ratio/aspect_ratio",
            "description": "宽高比，如 1:1 / 16:9。"
          },
          {
            "name": "resolution",
            "type": "string",
            "required": false,
            "mapping": "resolution",
            "description": "分辨率档位，如 1K / 2K / 4K。"
          },
          {
            "name": "quality",
            "type": "string",
            "required": false,
            "mapping": "quality",
            "description": "质量档位。"
          },
          {
            "name": "providerOptions",
            "type": "object",
            "required": false,
            "mapping": "provider-specific fields",
            "description": "插件命名空间内的厂商扩展字段。"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/images/create",
          "contentType": "application/json",
          "body": {
            "model": {
              "$ref": "request.model"
            },
            "prompt": {
              "$ref": "request.prompt"
            },
            "n": {
              "$omitEmpty": {
                "$if": {
                  "condition": {
                    "$gt": [
                      {
                        "$ref": "request.imageCount"
                      },
                      0
                    ]
                  },
                  "then": {
                    "$ref": "request.imageCount"
                  },
                  "else": 1
                }
              }
            },
            "size": {
              "$omitEmpty": {
                "$ref": "request.extra.size"
              }
            },
            "ratio": {
              "$omitEmpty": {
                "$coalesce": [
                  {
                    "$ref": "request.extra.ratio"
                  },
                  {
                    "$ref": "request.aspectRatio"
                  }
                ]
              }
            },
            "aspect_ratio": {
              "$omitEmpty": {
                "$coalesce": [
                  {
                    "$ref": "request.extra.ratio"
                  },
                  {
                    "$ref": "request.aspectRatio"
                  }
                ]
              }
            },
            "resolution": {
              "$omitEmpty": {
                "$coalesce": [
                  {
                    "$ref": "request.extra.resolution"
                  },
                  {
                    "$ref": "request.resolution"
                  }
                ]
              }
            },
            "quality": {
              "$omitEmpty": {
                "$ref": "request.quality"
              }
            },
            "images": {
              "$omitEmpty": {
                "$map": {
                  "from": {
                    "$sortByOrder": {
                      "$ref": "request.images"
                    }
                  },
                  "as": "media",
                  "in": {
                    "$ref": "media.value"
                  }
                }
              }
            },
            "references": {
              "$omitEmpty": {
                "$map": {
                  "from": {
                    "$sortByOrder": {
                      "$ref": "request.images"
                    }
                  },
                  "as": "media",
                  "in": {
                    "$ref": "media.value"
                  }
                }
              }
            }
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/images/tasks/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.data.task_id"
              },
              {
                "$ref": "response.data.id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.data.status"
              },
              {
                "$ref": "response.status"
              },
              {
                "$ref": "response.state"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.fail_reason"
              },
              {
                "$ref": "response.data.fail_reason"
              },
              {
                "$ref": "response.message"
              }
            ]
          },
          "images": {
            "$coalesce": [
              {
                "$ref": "response.result_urls"
              },
              {
                "$ref": "response.data.result_urls"
              },
              {
                "$ref": "response.result_url"
              },
              {
                "$ref": "response.data.result_url"
              },
              {
                "$ref": "response.url"
              },
              {
                "$ref": "response.data.url"
              },
              {
                "$ref": "response.data"
              }
            ]
          },
          "usage": {
            "$ref": "response.usage"
          },
          "errorPaths": [
            "error.code"
          ],
          "messagePaths": [
            "error.message",
            "fail_reason"
          ],
          "resultEphemeral": true
        }
      }
    ]
  }
}
```
<!-- YINGCE_MANIFEST_CONTRACT_END -->
