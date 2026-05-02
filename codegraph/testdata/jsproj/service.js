export class Base {
  constructor() {
    this.id = 0;
  }
}

export class Service extends Base {
  constructor(name) {
    super();
    this.name = name;
  }

  run() {
    return this.name + ': running';
  }

  stop() {
    cleanup();
  }
}

function cleanup() {}
