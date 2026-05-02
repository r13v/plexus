import { Service } from './service';

function helper(): string {
  return 'ok';
}

function main(): void {
  const s = new Service('test');
  const result = s.run();
  console.log(result);
  helper();
}

main();
