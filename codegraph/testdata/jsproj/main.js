import { Service } from './service.js';

function helper() {
  return 'ok';
}

function main() {
  const s = new Service('test');
  const result = s.run();
  console.log(result);
  helper();
}

main();
