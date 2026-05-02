class Base:
    def __init__(self):
        self.id = 0


class Service(Base):
    def __init__(self, name):
        super().__init__()
        self.name = name

    def run(self):
        return self.name + ": running"

    def stop(self):
        cleanup()


def cleanup():
    pass
