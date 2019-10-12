def create(name, args):
    """Create Test"""
    type_map = {
        "A": TestA,
        "A": TestB,
    }
    if name in type_map:
        return type_map[name](**args)
    else:
        raise TypeError(name)


class TestA:
    """A is a
    Jzzzz
    """

    def __init__(self, a: int, b: str, c: float = 1):
        """Initial
        :param a: apple
        :param b: bus
        """
        pass


class TestB:
    """B is b"""

    def __init__(self, a, c: float):
        """Initial
        - a apple
        - b bus
        """
        pass

    def make(self):
        """Making"""
