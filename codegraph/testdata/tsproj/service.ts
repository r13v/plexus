export interface Runner {
  run(): string;
}

export class Base {
  protected id = 0;
}

export class Service extends Base implements Runner {
  constructor(private name: string) {
    super();
  }

  run(): string {
    return this.name + ': running';
  }

  stop(): void {
    cleanup();
  }
}

function cleanup(): void {}
