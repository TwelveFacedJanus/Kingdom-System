from queue import Queue
from time import sleep
from typing import Self, Dict, Optional
from datetime import datetime, timedelta
import json
import requests
import grpc
from google.protobuf.timestamp_pb2 import Timestamp
import redis
import uuid

from proto import Authenticate_pb2 as auth_proto
from proto import Authenticate_pb2_grpc as auth_grpc
from oauth2_config import YANDEX_OAUTH_CONFIG
from environment import Environment

env = Environment(_path_to_env='../../../.env')

class TokenManager:
    def __init__(self: Self) -> None:
        try:
            self.redis_client = redis.Redis(
                host=env.redis_url,
                port=env.redis_port,
                password=env.redis_password,
                decode_responses=True,
                socket_timeout=5,
                socket_connect_timeout=5
            )
            self.redis_client.ping()
            print("Successfully connected to Redis")
        except redis.ConnectionError as e:
            print(f"Failed to connect to Redis: {str(e)}")
            raise
        except Exception as e:
            print(f"Unexpected error connecting to Redis: {str(e)}")
            raise
            
        self.token_prefix = "auth_token:"
        self.refresh_token_prefix = "refresh_token:"

    def store_tokens(self: Self, user_id: str, access_token: str, refresh_token: str, expires_in: int) -> None:
        """Store tokens in Redis with expiration."""

        self.redis_client.setex(
            f"{self.token_prefix}{access_token}",
            expires_in,
            user_id
        )
        
        self.redis_client.setex(
            f"{self.refresh_token_prefix}{refresh_token}",
            30 * 24 * 60 * 60,
            user_id
        )

    def validate_access_token(self: Self, access_token: str) -> Optional[str]:
        """Validate access token and return user_id if valid."""
        user_id = self.redis_client.get(f"{self.token_prefix}{access_token}")
        return user_id

    def validate_refresh_token(self: Self, refresh_token: str) -> Optional[str]:
        """Validate refresh token and return user_id if valid."""
        user_id = self.redis_client.get(f"{self.refresh_token_prefix}{refresh_token}")
        return user_id

    def revoke_tokens(self: Self, access_token: str, refresh_token: str) -> None:
        """Revoke both access and refresh tokens."""
        self.redis_client.delete(f"{self.token_prefix}{access_token}")
        self.redis_client.delete(f"{self.refresh_token_prefix}{refresh_token}")


class AuthenticateService(auth_grpc.AuthenticateServiceServicer):
    def __init__(self: Self) -> None:
        self.token_manager = TokenManager()
    
    def OAuth2Login(self, request: auth_proto.OAuth2LoginRequest, context: grpc.ServicerContext) -> auth_proto.OAuth2LoginResponse:
        """Generate OAuth2.0 login URL for Yandex authentication."""
        print('New OAuth2Login request.')

        state = request.state if request.state else str(uuid.uuid4())

        auth_url = (
            f"{YANDEX_OAUTH_CONFIG['authorize_url']}"
            f"?response_type=code"
            f"&client_id={YANDEX_OAUTH_CONFIG['client_id']}"
            f"&redirect_uri={YANDEX_OAUTH_CONFIG['redirect_uri']}"
            f"&scope={YANDEX_OAUTH_CONFIG['scope']}"
            f"&state={state}"
        )
        
        return auth_proto.OAuth2LoginResponse(auth_url=auth_url)
    
    def OAuth2Callback(self, request: auth_proto.OAuth2CallbackRequest, context: grpc.ServicerContext) -> auth_proto.OAuth2CallbackResponse:
        """Handle OAuth2.0 callback from Yandex."""
        print('New OAuth2Callback request.')
        
        try:
            token_response = requests.post(
                YANDEX_OAUTH_CONFIG['token_url'],
                data={
                    'grant_type': 'authorization_code',
                    'code': request.code,
                    'client_id': YANDEX_OAUTH_CONFIG['client_id'],
                    'client_secret': YANDEX_OAUTH_CONFIG['client_secret']
                }
            )
            token_data = token_response.json()
            
            if 'error' in token_data:
                context.set_code(grpc.StatusCode.INTERNAL)
                context.set_details(f"Token exchange failed: {token_data['error']}")
                return auth_proto.OAuth2CallbackResponse()
            
            access_token = token_data['access_token']
            refresh_token = token_data.get('refresh_token', '')
            expires_in = token_data.get('expires_in', 3600)

            user_response = requests.get(
                YANDEX_OAUTH_CONFIG['userinfo_url'],
                headers={'Authorization': f'OAuth {access_token}'}
            )
            user_data = user_response.json()

            self.token_manager.store_tokens(
                user_id=user_data['id'],
                access_token=access_token,
                refresh_token=refresh_token,
                expires_in=expires_in
            )

            user_profile = auth_proto.UserProfile(
                id=user_data['id'],
                email=user_data.get('default_email', ''),
                first_name=user_data.get('first_name', ''),
                last_name=user_data.get('last_name', ''),
                display_name=user_data.get('display_name', ''),
                avatar_url=user_data.get('default_avatar_id', '')
            )

            expires_at = Timestamp()
            expires_at.FromDatetime(
                datetime.utcnow() + timedelta(seconds=expires_in)
            )
            
            return auth_proto.OAuth2CallbackResponse(
                access_token=access_token,
                refresh_token=refresh_token,
                expires_at=expires_at,
                user_profile=user_profile
            )
            
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"OAuth2 callback failed: {str(e)}")
            return auth_proto.OAuth2CallbackResponse()

    def RefreshToken(self, request: auth_proto.RefreshTokenRequest, context: grpc.ServicerContext) -> auth_proto.RefreshTokenResponse:
        """Refresh the access token using a refresh token."""
        print('New RefreshToken request.')
        
        try:
            user_id = self.token_manager.validate_refresh_token(request.refresh_token)
            if not user_id:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details("Invalid or expired refresh token")
                return auth_proto.RefreshTokenResponse(
                    error="Invalid or expired refresh token"
                )

            token_response = requests.post(
                YANDEX_OAUTH_CONFIG['token_url'],
                data={
                    'grant_type': 'refresh_token',
                    'refresh_token': request.refresh_token,
                    'client_id': YANDEX_OAUTH_CONFIG['client_id'],
                    'client_secret': YANDEX_OAUTH_CONFIG['client_secret']
                }
            )
            token_data = token_response.json()
            
            if 'error' in token_data:
                context.set_code(grpc.StatusCode.INTERNAL)
                context.set_details(f"Token refresh failed: {token_data['error']}")
                return auth_proto.RefreshTokenResponse(
                    error=f"Token refresh failed: {token_data['error']}"
                )
            
            access_token = token_data['access_token']
            refresh_token = token_data.get('refresh_token', request.refresh_token)
            expires_in = token_data.get('expires_in', 3600)

            self.token_manager.store_tokens(
                user_id=user_id,
                access_token=access_token,
                refresh_token=refresh_token,
                expires_in=expires_in
            )

            expires_at = Timestamp()
            expires_at.FromDatetime(
                datetime.utcnow() + timedelta(seconds=expires_in)
            )
 
            token_data = auth_proto.RefreshTokenResponseData(
                auth_token=access_token,
                refresh_token=refresh_token,
                expires_at=expires_at
            )
            
            return auth_proto.RefreshTokenResponse(token=token_data)
            
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Token refresh failed: {str(e)}")
            return auth_proto.RefreshTokenResponse(
                error=f"Token refresh failed: {str(e)}"
            )

    def GetProfileByToken(self, request: auth_proto.GetProfileByTokenRequest, context: grpc.ServicerContext) -> auth_proto.UserProfile:
        """Get user profile information using an access token."""
        print('New GetProfileByToken request.')
        
        try:
            user_id = self.token_manager.validate_access_token(request.access_token)
            if not user_id:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details("Invalid or expired access token")
                return auth_proto.UserProfile()

            user_response = requests.get(
                YANDEX_OAUTH_CONFIG['userinfo_url'],
                headers={'Authorization': f'OAuth {request.access_token}'}
            )
            
            if user_response.status_code != 200:
                context.set_code(grpc.StatusCode.UNAUTHENTICATED)
                context.set_details("Invalid or expired access token")
                return auth_proto.UserProfile()
            
            user_data = user_response.json()

            return auth_proto.UserProfile(
                id=user_data['id'],
                email=user_data.get('default_email', ''),
                first_name=user_data.get('first_name', ''),
                last_name=user_data.get('last_name', ''),
                display_name=user_data.get('display_name', ''),
                avatar_url=user_data.get('default_avatar_id', '')
            )
            
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Failed to get user profile: {str(e)}")
            return auth_proto.UserProfile()

