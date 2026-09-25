package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"infinite-canvas/backend/internal/kernel"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"infinite-canvas/backend/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const SessionCookieName = "open_ai_canvas_session"

const sessionMaxAge = 30 * 24 * time.Hour

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

// AuthError 保留为兼容别名；跨认证域的新代码应直接使用 AppError。
type AuthError = kernel.AppError

type RegisterRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	EmailCode   string `json:"emailCode"`
	Phone       string `json:"phone"`
	SmsCode     string `json:"smsCode"`
	Channel     string `json:"channel"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	InviteCode  string `json:"inviteCode"`
	Host        string `json:"-"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type PublicAuthSettings struct {
	FirstUser           bool   `json:"firstUser"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
	LinuxDOEnabled      bool   `json:"linuxdoEnabled"`
	EmailEnabled        bool   `json:"emailEnabled"`
	EmailCodeRequired   bool   `json:"emailCodeRequired"`
	InviteRequired      bool   `json:"inviteRequired"`
	InviteLocked        bool   `json:"inviteLocked"`
	InviteDisplayName   string `json:"inviteDisplayName,omitempty"`
	SmsEnabled          bool   `json:"smsEnabled"`
	SmsCodeRequired     bool   `json:"smsCodeRequired"`
}

type AuthSessionResult struct {
	User       AuthUser `json:"user"`
	Session    string   `json:"session"`
	MaxAgeSecs int      `json:"maxAgeSecs"`
}

type AuthUser struct {
	model.User
	AvatarURL        string `json:"avatarUrl,omitempty"`
	IdentityProvider string `json:"identityProvider,omitempty"`
	IdentityID       string `json:"identityId,omitempty"`
	IdentityUsername string `json:"identityUsername,omitempty"`
}

func (s *Service) PublicAuthSettings(host string) (*PublicAuthSettings, error) {
	count, err := s.repo.UserCount()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return &PublicAuthSettings{FirstUser: true, RegistrationEnabled: true, LinuxDOEnabled: false}, nil
	}
	allowed, streamer, err := s.registrationAllowed(host, "")
	if err != nil {
		return nil, err
	}
	emailEnabled, err := s.EmailEnabled()
	if err != nil {
		return nil, err
	}
	smsEnabled, err := s.SMSEnabled()
	if err != nil {
		return nil, err
	}
	out := &PublicAuthSettings{FirstUser: false, RegistrationEnabled: allowed, LinuxDOEnabled: s.LinuxDOEnabled(), EmailEnabled: emailEnabled, EmailCodeRequired: true, SmsEnabled: smsEnabled, SmsCodeRequired: true}
	if streamer != nil && streamer.Status == model.StreamerStatusActive {
		hostStreamer, err := s.host.StreamerByHost(host)
		if err != nil {
			return nil, err
		}
		if hostStreamer != nil && hostStreamer.Status == model.StreamerStatusActive {
			out.InviteRequired = true
			out.InviteLocked = true
			out.InviteDisplayName = hostStreamer.DisplayName
		}
	}
	return out, nil
}

func (s *Service) Register(req RegisterRequest) (*AuthSessionResult, error) {
	username := NormalizeUsername(req.Username)
	email := NormalizeEmail(req.Email)
	phone := kernel.NormalizeChinaMobile(req.Phone)
	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	displayName := NormalizeDisplayName(req.DisplayName, username)
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}
	if email != "" {
		if err := ValidateEmail(email); err != nil {
			return nil, err
		}
	}
	if phone != "" && !kernel.IsChinaMobile(phone) {
		return nil, kernel.BadAuthRequest("请输入中国大陆 11 位手机号")
	}
	if channel == "" {
		if phone != "" {
			channel = "sms"
		} else {
			channel = "email"
		}
	}
	if channel != "email" && channel != "sms" {
		return nil, kernel.BadAuthRequest("请选择邮箱或短信注册")
	}
	s.registrationMu.Lock()
	defer s.registrationMu.Unlock()
	count, err := s.repo.UserCount()
	if err != nil {
		return nil, err
	}
	var verifiedEmail *model.EmailVerificationCode
	var verifiedSMS *model.SmsVerificationCode
	var inviteStreamer *model.Streamer
	if count > 0 {
		allowed, streamer, err := s.registrationAllowed(req.Host, req.InviteCode)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, kernel.Forbidden("管理员未开放新用户注册")
		}
		hostStreamer, err := s.host.StreamerByHost(req.Host)
		if err != nil {
			return nil, err
		}
		if hostStreamer != nil && hostStreamer.Status == model.StreamerStatusActive {
			inviteStreamer = hostStreamer
		} else {
			inviteStreamer = streamer
		}
		if channel == "sms" {
			smsEnabled, err := s.SMSEnabled()
			if err != nil {
				return nil, err
			}
			if !smsEnabled {
				return nil, kernel.Forbidden("管理员尚未配置注册短信")
			}
			if phone == "" {
				return nil, kernel.BadAuthRequest("请输入手机号")
			}
			verifiedSMS, err = s.VerifyRegistrationSMSCode(phone, req.SmsCode)
			if err != nil {
				return nil, err
			}
			email = ""
		} else {
			if email == "" {
				return nil, kernel.BadAuthRequest("请输入邮箱")
			}
			if err := s.validateRegistrationEmailDomain(email); err != nil {
				return nil, err
			}
			verifiedEmail, err = s.VerifyRegistrationEmailCode(email, req.EmailCode)
			if err != nil {
				return nil, err
			}
			phone = ""
		}
	}
	if _, err := s.repo.UserByUsername(username); err == nil {
		return nil, kernel.BadAuthRequest("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if email != "" {
		if _, err := s.repo.UserByEmail(email); err == nil {
			return nil, kernel.BadAuthRequest("邮箱已被注册")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if phone != "" {
		if _, err := s.repo.UserByPhone(phone); err == nil {
			return nil, kernel.BadAuthRequest("手机号已被注册")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user := model.User{
		ID:           kernel.NewID(),
		Username:     username,
		Email:        email,
		Phone:        phone,
		DisplayName:  displayName,
		Role:         model.UserRoleUser,
		Status:       model.UserStatusActive,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if count == 0 {
		user.Role = model.UserRoleAdmin
	}
	if verifiedEmail != nil {
		if err := s.repo.CreateUserWithEmailVerification(&user, verifiedEmail.ID, time.Now()); err != nil {
			return nil, err
		}
	} else if verifiedSMS != nil {
		if err := s.repo.CreateUserWithSmsVerification(&user, verifiedSMS.ID, time.Now()); err != nil {
			return nil, err
		}
	} else if err := s.repo.Create(&user); err != nil {
		return nil, err
	}
	if err := s.host.EnsureSignupBonus(user.ID); err != nil {
		return nil, err
	}
	if inviteStreamer != nil {
		if err := s.host.BindUserStreamerInvite(user.ID, inviteStreamer.ID); err != nil {
			return nil, err
		}
	}
	return s.createAuthSession(&user)
}

func (s *Service) Login(req LoginRequest) (*AuthSessionResult, error) {
	account := strings.TrimSpace(req.Username)
	user, err := s.repo.UserByAccount(account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.Unauthorized("用户名、邮箱、手机号或密码不正确")
		}
		return nil, err
	}
	if user.Status != model.UserStatusActive {
		return nil, kernel.Forbidden("该账号已被禁用")
	}
	if !verifyPassword(req.Password, user.PasswordHash) {
		return nil, kernel.Unauthorized("用户名、邮箱、手机号或密码不正确")
	}
	now := time.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	if err := s.repo.Save(user); err != nil {
		return nil, err
	}
	if err := s.host.EnsureSignupBonus(user.ID); err != nil {
		return nil, err
	}
	s.host.RecordActivity(user.ID, "login", 1)
	return s.createAuthSession(user)
}

func (s *Service) Logout(cookieValue string) error {
	sessionID, _ := parseSessionCookie(cookieValue)
	if sessionID == "" {
		return nil
	}
	return s.repo.DeleteAuthSession(sessionID)
}

func (s *Service) CurrentUser(cookieValue string) (*model.User, error) {
	sessionID, token := parseSessionCookie(cookieValue)
	if sessionID == "" || token == "" {
		return nil, kernel.Unauthorized("请先登录")
	}
	session, err := s.repo.AuthSession(sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, kernel.Unauthorized("登录状态已失效")
		}
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) || session.TokenHash != HashToken(token) {
		if cleanupErr := s.repo.DeleteAuthSession(sessionID); cleanupErr != nil {
			log.Printf("expired auth session cleanup failed: session_id=%s error=%v", sessionID, cleanupErr)
		}
		return nil, kernel.Unauthorized("登录状态已失效")
	}
	user, err := s.repo.User(session.UserID)
	if err != nil {
		return nil, err
	}
	if user.Status != model.UserStatusActive {
		return nil, kernel.Forbidden("该账号已被禁用")
	}
	return user, nil
}

// 认证响应只补充当前用户自己的第三方公开身份，不把身份表或密钥字段暴露给其他列表接口。
func (s *Service) PublicAuthUser(user *model.User) (AuthUser, error) {
	result := AuthUser{User: *user}
	identity, err := s.repo.UserIdentityForUser(user.ID, "linuxdo")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	if err != nil {
		return AuthUser{}, err
	}
	result.AvatarURL = identity.AvatarURL
	result.IdentityProvider = identity.Provider
	result.IdentityID = identity.Subject
	result.IdentityUsername = identity.ProviderUsername
	return result, nil
}

func (s *Service) createAuthSession(user *model.User) (*AuthSessionResult, error) {
	publicUser, err := s.PublicAuthUser(user)
	if err != nil {
		return nil, err
	}
	token := RandomToken()
	now := time.Now()
	session := model.AuthSession{
		ID:        kernel.NewID(),
		UserID:    user.ID,
		TokenHash: HashToken(token),
		ExpiresAt: now.Add(sessionMaxAge),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(&session); err != nil {
		return nil, err
	}
	return &AuthSessionResult{User: publicUser, Session: session.ID + "." + token, MaxAgeSecs: int(sessionMaxAge.Seconds())}, nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func verifyPassword(password string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func RandomToken() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return kernel.NewID() + kernel.NewID()
	}
	return hex.EncodeToString(b[:])
}

func parseSessionCookie(value string) (string, string) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func NormalizeUsername(value string) string {
	return strings.TrimSpace(value)
}

func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeDisplayName(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	runes := []rune(value)
	if len(runes) > 40 {
		value = string(runes[:40])
	}
	return value
}

func ValidateUsername(value string) error {
	if !usernamePattern.MatchString(value) {
		return kernel.BadAuthRequest("用户名需为 3-32 位字母、数字、下划线或连字符")
	}
	return nil
}

func ValidatePassword(value string) error {
	if len([]rune(value)) < 8 {
		return kernel.BadAuthRequest("密码至少 8 位")
	}
	return nil
}

func ValidateEmail(value string) error {
	if _, err := mail.ParseAddress(value); err != nil {
		return kernel.BadAuthRequest("邮箱格式不正确")
	}
	return nil
}
