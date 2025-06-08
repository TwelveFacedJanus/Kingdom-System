from environment import Environment
from typing import Dict

env = Environment(_path_to_env='../../../.env')

YANDEX_OAUTH_CONFIG: Dict[str, str] = {
    'client_id': env.yandex_client_id,
    'client_secret': env.yandex_client_secret,
    'authorize_url': 'https://oauth.yandex.com/authorize',
    'token_url': 'https://oauth.yandex.com/token',
    'userinfo_url': 'https://login.yandex.ru/info',
    'redirect_uri': env.yandex_redirect_uri,
    'scope': 'login:info login:email login:avatar'
} 