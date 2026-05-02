from service import Service


def helper():
    return "ok"


def main():
    s = Service("test")
    result = s.run()
    print(result)
    helper()
