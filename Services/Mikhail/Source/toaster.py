from typing import Self, Dict, Callable, Any

class Toster:
    _tests: Dict[str, Callable[..., Any]] = {}

    @classmethod
    def register(cls, f: Callable[..., Any]) -> Callable[..., Any]:
        cls._tests[f.__name__] = f
    
    @classmethod
    def run_tests(cls) -> None:
        print('='*10 + f' TEST COUNT {len(cls._tests)} ' + '='*10)
        c = 1
        for fname, func in cls._tests.items():
            print(f"- {c}/{len(cls._tests)} [ {fname} ]", end=':')
            try:
                if func() == True:
                    print(" OK ")
                else:
                    print(" FAILED ")
            except Exception as ex:
                print(f" EXCEPTION -- {ex}")



