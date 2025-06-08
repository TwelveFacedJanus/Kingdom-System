package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	authenticate_proto "Mikhail/gen/proto"

	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TokenInfo represents the information stored with a refresh token
type TokenInfo struct {
	UserID       string
	PhoneNumber  string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	IsYandexUser bool
	YandexToken  *oauth2.Token
}

// TokenStorage defines the interface for token storage
type TokenStorage interface {
	StoreRefreshToken(token string, info TokenInfo) error
	GetTokenInfo(token string) (*TokenInfo, error)
	DeleteToken(token string) error
}

// InMemoryTokenStorage implements TokenStorage interface using in-memory storage
type InMemoryTokenStorage struct {
	tokens      map[string]TokenInfo
	mu          sync.RWMutex
	stopCleanup chan struct{}
	maxSize     int
}

func NewInMemoryTokenStorage() *InMemoryTokenStorage {
	storage := &InMemoryTokenStorage{
		tokens:      make(map[string]TokenInfo),
		stopCleanup: make(chan struct{}),
		maxSize:     10000, // Limit to 10k tokens
	}
	// Start cleanup goroutine
	go storage.cleanupExpiredTokens()
	return storage
}

func (s *InMemoryTokenStorage) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for token, info := range s.tokens {
				if now.After(info.ExpiresAt) {
					delete(s.tokens, token)
				}
			}
			s.mu.Unlock()
		case <-s.stopCleanup:
			return
		}
	}
}

func (s *InMemoryTokenStorage) Close() {
	close(s.stopCleanup)
}

func (s *InMemoryTokenStorage) StoreRefreshToken(token string, info TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we're at capacity
	if len(s.tokens) >= s.maxSize {
		return fmt.Errorf("token storage is at capacity")
	}

	// Validate token info
	if info.UserID == "" || info.PhoneNumber == "" {
		return fmt.Errorf("invalid token info: missing required fields")
	}

	s.tokens[token] = info
	return nil
}

func (s *InMemoryTokenStorage) GetTokenInfo(token string) (*TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if info, exists := s.tokens[token]; exists {
		return &info, nil
	}
	return nil, fmt.Errorf("token not found")
}

func (s *InMemoryTokenStorage) DeleteToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
	return nil
}

type AuthServer struct {
	authenticate_proto.UnimplementedAuthenticateServiceServer
	tokenStorage TokenStorage
	// Add channels for async operations
	tokenUpdateChan chan *tokenUpdateRequest
	// Add rate limiter
	rateLimiter *RateLimiter
}

type tokenUpdateRequest struct {
	oldToken string
	newToken string
	info     TokenInfo
}

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	window   time.Duration
	limit    int
}

func NewRateLimiter(window time.Duration, limit int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		window:   window,
		limit:    limit,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Clean up old requests
	validRequests := make([]time.Time, 0)
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	rl.requests[key] = validRequests

	// Check if under limit
	if len(validRequests) >= rl.limit {
		return false
	}

	// Add new request
	rl.requests[key] = append(validRequests, now)
	return true
}

func NewAuthServer() *AuthServer {
	// Get Redis configuration from environment
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	// Decode hex-encoded encryption key
	hexKey := getEnv("REDIS_ENCRYPTION_KEY", "your-32-byte-encryption-key-here")
	encryptionKey, err := hex.DecodeString(hexKey)
	if err != nil {
		log.Fatalf("Failed to decode encryption key: %v", err)
	}

	// Create Redis storage
	storage, err := NewRedisTokenStorage(redisURL, redisPassword, encryptionKey)
	if err != nil {
		log.Fatalf("Failed to create Redis storage: %v", err)
	}

	server := &AuthServer{
		tokenStorage:    storage,
		tokenUpdateChan: make(chan *tokenUpdateRequest, 100),
		rateLimiter:     NewRateLimiter(1*time.Minute, 60),
	}
	go server.tokenUpdateWorker()
	return server
}

func (s *AuthServer) tokenUpdateWorker() {
	for update := range s.tokenUpdateChan {
		// Use context with timeout for token operations
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Store new token
		if err := s.tokenStorage.StoreRefreshToken(update.newToken, update.info); err != nil {
			log.Printf("Failed to store new refresh token: %v", err)
			cancel()
			continue
		}

		// Delete old token
		if err := s.tokenStorage.DeleteToken(update.oldToken); err != nil {
			log.Printf("Warning: Failed to delete old refresh token: %v", err)
		}

		cancel()
	}
}

// Ensure AuthServer implements the interface
var _ authenticate_proto.AuthenticateServiceServer = (*AuthServer)(nil)

// YandexUserProfile represents the user profile data from Yandex
type YandexUserProfile struct {
	ID          string `json:"id"`
	Email       string `json:"default_email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"default_avatar_id"`
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *authenticate_proto.RefreshTokenRequest) (*authenticate_proto.RefreshTokenResponse, error) {
	// Add rate limiting
	if !s.rateLimiter.Allow(req.RefreshToken) {
		return &authenticate_proto.RefreshTokenResponse{
			Response: &authenticate_proto.RefreshTokenResponse_Error{
				Error: "rate limit exceeded",
			},
		}, nil
	}

	// Validate input
	if req.RefreshToken == "" {
		return &authenticate_proto.RefreshTokenResponse{
			Response: &authenticate_proto.RefreshTokenResponse_Error{
				Error: "refresh token is required",
			},
		}, nil
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get token info from storage
	tokenInfo, err := s.tokenStorage.GetTokenInfo(req.RefreshToken)
	if err != nil {
		return &authenticate_proto.RefreshTokenResponse{
			Response: &authenticate_proto.RefreshTokenResponse_Error{
				Error: "invalid refresh token",
			},
		}, nil
	}

	// Check if token is expired
	if time.Now().After(tokenInfo.ExpiresAt) {
		s.tokenStorage.DeleteToken(req.RefreshToken)
		return &authenticate_proto.RefreshTokenResponse{
			Response: &authenticate_proto.RefreshTokenResponse_Error{
				Error: "refresh token expired",
			},
		}, nil
	}

	// Create a channel for Yandex token refresh
	yandexTokenChan := make(chan *oauth2.Token, 1)
	yandexErrorChan := make(chan error, 1)

	// Refresh Yandex OAuth token if it's a Yandex user
	if tokenInfo.IsYandexUser && tokenInfo.YandexToken != nil {
		go func() {
			log.Printf("Attempting to refresh Yandex token with: AccessToken=%s, RefreshToken=%s, ExpiresAt=%v",
				tokenInfo.YandexToken.AccessToken, tokenInfo.YandexToken.RefreshToken, tokenInfo.YandexToken.Expiry)
			newToken, err := refreshYandexToken(ctx, tokenInfo.YandexToken)
			if err != nil {
				yandexErrorChan <- err
				return
			}
			yandexTokenChan <- newToken
		}()
	} else if tokenInfo.IsYandexUser {
		log.Printf("Yandex user but no Yandex token found")
		return &authenticate_proto.RefreshTokenResponse{
			Response: &authenticate_proto.RefreshTokenResponse_Error{
				Error: "invalid Yandex token state",
			},
		}, nil
	}

	// For Yandex users, use Yandex token values
	var newAuthToken, newRefreshToken string
	var expiry time.Time
	var newYandexToken *oauth2.Token

	if tokenInfo.IsYandexUser {
		select {
		case err := <-yandexErrorChan:
			log.Printf("Failed to refresh Yandex token: %v", err)
			return &authenticate_proto.RefreshTokenResponse{
				Response: &authenticate_proto.RefreshTokenResponse_Error{
					Error: "failed to refresh Yandex token",
				},
			}, nil
		case newYandexToken = <-yandexTokenChan:
			if newYandexToken == nil {
				log.Printf("New Yandex token is nil for Yandex user")
				return &authenticate_proto.RefreshTokenResponse{
					Response: &authenticate_proto.RefreshTokenResponse_Error{
						Error: "failed to refresh Yandex token",
					},
				}, nil
			}
			log.Printf("New Yandex token details: AccessToken=%s, RefreshToken=%s, ExpiresAt=%v",
				newYandexToken.AccessToken, newYandexToken.RefreshToken, newYandexToken.Expiry)

			newAuthToken = newYandexToken.AccessToken
			newRefreshToken = newYandexToken.RefreshToken
			expiry = newYandexToken.Expiry
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		// For non-Yandex users, generate new tokens
		newAuthToken = generate_auth_token(tokenInfo.PhoneNumber, "")
		newRefreshToken = generate_refresh_token()
		expiry = time.Now().Add(24 * time.Hour)
	}

	// Create new token info
	newTokenInfo := TokenInfo{
		UserID:       tokenInfo.UserID,
		PhoneNumber:  tokenInfo.PhoneNumber,
		CreatedAt:    time.Now(),
		ExpiresAt:    expiry,
		IsYandexUser: tokenInfo.IsYandexUser,
		YandexToken:  newYandexToken,
	}

	// Send token update request to worker
	select {
	case s.tokenUpdateChan <- &tokenUpdateRequest{
		oldToken: req.RefreshToken,
		newToken: newRefreshToken,
		info:     newTokenInfo,
	}:
		// Token update request sent successfully
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &authenticate_proto.RefreshTokenResponse{
		Response: &authenticate_proto.RefreshTokenResponse_Token{
			Token: &authenticate_proto.RefreshTokenResponseData{
				AuthToken:    newAuthToken,
				RefreshToken: newRefreshToken,
				ExpiresAt:    timestamppb.New(expiry),
			},
		},
	}, nil
}

// refreshYandexToken refreshes a Yandex OAuth token by directly making an HTTP request.
func refreshYandexToken(ctx context.Context, yandexToken *oauth2.Token) (*oauth2.Token, error) {
	if yandexToken.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token provided in yandexToken")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", yandexToken.RefreshToken)
	data.Set("client_id", oauth2Config.ClientID)
	data.Set("client_secret", oauth2Config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", oauth2Config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(oauth2Config.ClientID, oauth2Config.ClientSecret)

	// Use a client with connection pooling
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yandex token refresh failed with status code %d: %s", resp.StatusCode, resp.Status)
	}

	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Yandex token response: %w", err)
	}

	// Yandex might not return a new refresh token if the existing one is still valid
	newRefreshToken := tokenResponse.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = yandexToken.RefreshToken
	}

	return &oauth2.Token{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: newRefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second),
	}, nil
}

func (s *AuthServer) SignOut(ctx context.Context, req *authenticate_proto.SignOutRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *AuthServer) SignUp(ctx context.Context, req *authenticate_proto.SignUpRequest) (*authenticate_proto.SignUpResponse, error) {
	// Implement your signup logic here
	authToken := generate_auth_token(req.PhoneNumber, req.PasswordHash)
	refreshToken := generate_refresh_token()

	// Store token information
	tokenInfo := TokenInfo{
		UserID:       req.PhoneNumber, // Using phone number as UserID for now
		PhoneNumber:  req.PhoneNumber,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour), // 30 days
		IsYandexUser: false,
		YandexToken:  nil,
	}
	if err := s.tokenStorage.StoreRefreshToken(refreshToken, tokenInfo); err != nil {
		log.Printf("Failed to store refresh token during signup: %v", err)
		return &authenticate_proto.SignUpResponse{
			Response: &authenticate_proto.SignUpResponse_Error{
				Error: "failed to store refresh token",
			},
		}, nil
	}

	return &authenticate_proto.SignUpResponse{
		Response: &authenticate_proto.SignUpResponse_Token{
			Token: &authenticate_proto.RefreshTokenResponseData{
				AuthToken:    authToken,
				RefreshToken: refreshToken,
				ExpiresAt:    timestamppb.New(time.Now().Add(24 * time.Hour)), // Access token expires in 24 hours
			},
		},
	}, nil
}

func (s *AuthServer) SignIn(ctx context.Context, req *authenticate_proto.SignInRequest) (*authenticate_proto.SignInResponse, error) {
	// Implement your signin logic here
	authToken := generate_auth_token(req.PhoneNumber, req.PasswordHash)
	refreshToken := generate_refresh_token()

	// Store token information
	tokenInfo := TokenInfo{
		UserID:       req.PhoneNumber, // Using phone number as UserID for now
		PhoneNumber:  req.PhoneNumber,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour), // 30 days
		IsYandexUser: false,
		YandexToken:  nil,
	}
	if err := s.tokenStorage.StoreRefreshToken(refreshToken, tokenInfo); err != nil {
		log.Printf("Failed to store refresh token during signin: %v", err)
		return &authenticate_proto.SignInResponse{
			Response: &authenticate_proto.SignInResponse_Error{
				Error: "failed to store refresh token",
			},
		}, nil
	}

	return &authenticate_proto.SignInResponse{
		Response: &authenticate_proto.SignInResponse_Token{
			Token: &authenticate_proto.RefreshTokenResponseData{
				AuthToken:    authToken,
				RefreshToken: refreshToken,
				ExpiresAt:    timestamppb.New(time.Now().Add(24 * time.Hour)), // Access token expires in 24 hours
			},
		},
	}, nil
}

func (s *AuthServer) OAuth2Login(ctx context.Context, req *authenticate_proto.OAuth2LoginRequest) (*authenticate_proto.OAuth2LoginResponse, error) {
	log.Printf("Received OAuth2 login request with state: %s", req.State)

	url := GetOAuth2LoginURL(req.State)
	if url == "" {
		log.Printf("Failed to generate OAuth2 login URL")
		return nil, fmt.Errorf("failed to generate OAuth2 login URL")
	}

	log.Printf("Generated OAuth2 login URL: %s", url)
	return &authenticate_proto.OAuth2LoginResponse{
		AuthUrl: url,
	}, nil
}

func (s *AuthServer) OAuth2Callback(ctx context.Context, req *authenticate_proto.OAuth2CallbackRequest) (*authenticate_proto.OAuth2CallbackResponse, error) {
	log.Printf("Received OAuth2 callback with code: %s", req.Code)

	// Exchange code for token
	token, err := ExchangeCode(ctx, req.Code)
	if err != nil {
		log.Printf("Failed to exchange code for token: %v", err)
		return nil, fmt.Errorf("failed to exchange code for token: %v", err)
	}
	log.Printf("Yandex token received: AccessToken=%s, RefreshToken=%s, ExpiresAt=%v", token.AccessToken, token.RefreshToken, token.Expiry)

	// Fetch user profile from Yandex
	profile, err := fetchYandexUserProfile(ctx, token.AccessToken)
	if err != nil {
		log.Printf("Failed to fetch user profile: %v", err)
		return nil, fmt.Errorf("failed to fetch user profile: %v", err)
	}

	// Generate our own refresh token
	refreshToken := generate_refresh_token()

	// Store token information
	tokenInfo := TokenInfo{
		UserID:       profile.ID,
		PhoneNumber:  profile.Email, // Using email as phone number for Yandex users
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour), // 30 days
		IsYandexUser: true,
		YandexToken:  token,
	}
	if err := s.tokenStorage.StoreRefreshToken(refreshToken, tokenInfo); err != nil {
		log.Printf("Failed to store refresh token: %v", err)
		return nil, fmt.Errorf("failed to store refresh token: %v", err)
	}

	log.Printf("Successfully authenticated user: %s", profile.DisplayName)
	return &authenticate_proto.OAuth2CallbackResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    timestamppb.New(token.Expiry),
		UserProfile: &authenticate_proto.UserProfile{
			Id:          profile.ID,
			Email:       profile.Email,
			FirstName:   profile.FirstName,
			LastName:    profile.LastName,
			DisplayName: profile.DisplayName,
			AvatarUrl:   fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-200", profile.AvatarURL),
		},
	}, nil
}

func (s *AuthServer) GetMe(ctx context.Context, _ *emptypb.Empty) (*authenticate_proto.UserProfile, error) {
	// Get the authorization token from the context
	authToken := ctx.Value("auth_token")
	if authToken == nil {
		return nil, fmt.Errorf("unauthorized: no auth token provided")
	}

	// Fetch user profile from Yandex using the auth token
	profile, err := fetchYandexUserProfile(ctx, authToken.(string))
	if err != nil {
		log.Printf("Failed to fetch user profile: %v", err)
		return nil, fmt.Errorf("failed to fetch user profile: %v", err)
	}

	return &authenticate_proto.UserProfile{
		Id:          profile.ID,
		Email:       profile.Email,
		FirstName:   profile.FirstName,
		LastName:    profile.LastName,
		DisplayName: profile.DisplayName,
		AvatarUrl:   fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-200", profile.AvatarURL),
	}, nil
}

func (s *AuthServer) GetProfileByToken(ctx context.Context, req *authenticate_proto.GetProfileByTokenRequest) (*authenticate_proto.UserProfile, error) {
	if req.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	// Fetch user profile from Yandex using the provided access token
	profile, err := fetchYandexUserProfile(ctx, req.AccessToken)
	if err != nil {
		log.Printf("Failed to fetch user profile: %v", err)
		return nil, fmt.Errorf("failed to fetch user profile: %v", err)
	}

	return &authenticate_proto.UserProfile{
		Id:          profile.ID,
		Email:       profile.Email,
		FirstName:   profile.FirstName,
		LastName:    profile.LastName,
		DisplayName: profile.DisplayName,
		AvatarUrl:   fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-200", profile.AvatarURL),
	}, nil
}

func fetchYandexUserProfile(ctx context.Context, accessToken string) (*YandexUserProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://login.yandex.ru/info", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "OAuth "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch user profile: status code %d", resp.StatusCode)
	}

	var profile YandexUserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %v", err)
	}

	return &profile, nil
}

func (s *AuthServer) Close() error {
	// Close token update channel
	close(s.tokenUpdateChan)

	// Close Redis storage
	if redisStorage, ok := s.tokenStorage.(*RedisTokenStorage); ok {
		return redisStorage.Close()
	}
	return nil
}
