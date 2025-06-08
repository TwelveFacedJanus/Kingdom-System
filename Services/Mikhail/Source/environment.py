from typing import Self
from dotenv import load_dotenv
import os

class Environment:
    def __init__(self: Self, _path_to_env: str = '.env') -> None:
        assert type(_path_to_env) == str, 'Path to .ENV must be string' 
        assert _path_to_env != '', 'Path to .ENV cannot be nullable string.'
        assert os.path.exists(_path_to_env) != False, f'Cannot find f{_path_to_env}'

        load_dotenv(_path_to_env)

        self.redis_host: str = os.getenv('REDIS_URL')
        self.redis_port: int = int(os.getenv('REDIS_PORT'))
        self.redis_password: str = os.getenv('REDIS_PASSWORD')
        self.redis_encrypted_key: str = os.getenv('REDIS_ENCRYPTED_KEY')
        self.mikhail_max_workers: int = int(os.getenv('MIKHAIL_MAX_WORKERS'))

        assert self.redis_host != '', 'Redis url is nullable?'
        assert self.redis_password != '', 'Redis password is nullable?'
        assert self.redis_encrypted_key != '', 'Redis encrypted key is nullable?'
        assert type(self.mikhail_max_workers) == int, 'Mikhail max workers must be integer.'

        # Yandex OAuth2.0 Configuration
        self.yandex_client_id: str = os.getenv('YANDEX_OAUTH_CLIENT_ID', '')
        self.yandex_client_secret: str = os.getenv('YANDEX_OAUTH_CLIENT_SECRET', '')
        self.yandex_redirect_uri: str = os.getenv('OAUTH_REDIRECTION_URL', '')

        # Redis Configuration
        self.redis_url: str = os.getenv('REDIS_URL', '')
        self.redis_encryption_key: str = os.getenv('REDIS_ENCRYPTION_KEY', '')
        self.redis_password: str = os.getenv('REDIS_PASSWORD', '')
        # Service Configuration
        self.port: int = int(os.getenv('PORT', '50051'))
        self.environment: str = os.getenv('ENVIRONMENT', 'development')

        print('Environment loaded!')


