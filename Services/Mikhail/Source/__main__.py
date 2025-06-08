from typing import Self, Dict, Callable, Any
from environment import Environment
from time import sleep

import grpc
from concurrent import futures

from proto import Authenticate_pb2_grpc as AuthenticatePackage
from authenticate_service import *
env = Environment(_path_to_env='../../../.env')


class Server:
    def __init__(self: Self,
                 _port: int = 5000,
                 _host: str = '[::]',
                 _max_workers: int = 10
                 ) -> None:
        assert type(_port) == int, 'Port must be integer'
        assert _port > 2000, 'Port cannot be lower that 2000'
        assert type(_host) == str, 'Host must be string value'
        assert _host != '', 'Host cannot be nullable'
        assert type(_max_workers) == int, 'Max workers is string. Are u kidding?'

        self._port: int = _port
        self._host: str = _host

        self._server: object = grpc.server(futures.ThreadPoolExecutor(max_workers=_max_workers))
        AuthenticatePackage.add_AuthenticateServiceServicer_to_server(AuthenticateService(), self._server)

    def serve(self: Self):
        print('Starting server...')

        self._server.add_insecure_port(f'{self._host}:{self._port}')
        self._server.start()

        print(f'Listening on {self._host}:{self._port}')
        print('Press CTRL+C to stop...')

        try:
            self._server.wait_for_termination()
        except KeyboardInterrupt:
            self._server.stop(None)
            print('Server is stopped')

s = Server(_port=env.port, _host='[::]', _max_workers=env.mikhail_max_workers)
s.serve()