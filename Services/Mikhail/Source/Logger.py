"""
    Filename:      /Services/Mikhail/Source/Logger.py
    Description:   This module provides simple logger without external dependencies.
                   
                   Usage example:
                   ```python
                   Logger.initialize(logtype='ALL')
                   Logger.log("Hi. Im log!")
                   ```

    Created at:    08.06.2025.
    Updated at:    08.06.2025.
    License:       BSD-3 Clause License.
"""

import os
from datetime import datetime
from typing import List, Union
import sys

LOGTYPES: List[str] = ['ALL', 'DEBUG', 'INFO', 'NOTHING']
TEMPLATE: str = "[ DATETIME ]:=:[ CALLERNAME ]:=[ LOGTYPE ]: "

class Logger:
    logtype: str
    template: str

    @classmethod
    def initialize(cls, logtype: str = 'ALL', template: str = TEMPLATE) -> Union[str, None]:
        if logtype not in LOGTYPES:
            return 'Log type not exists'
        cls.logtype: str = logtype
        cls.template: str = template

    @classmethod
    def log(cls, message: str = '') -> None:
        message: str = TEMPLATE.replace('DATETIME', str(datetime.now())).replace('CALLERNAME', sys._getframe(1).f_code.co_name).replace('LOGTYPE', cls.logtype)
        print(message)
