const { actor } = require('kar')

async function main () {
  let x = 0
  const a = actor.proxy('Foo', 'myInstance')
  for (let i = 0; i < 5000; i++) {
    x = await actor.call(a, 'incr', x)
    console.log(i, '->', x)
  }
  console.log('=>', x)
}

main()
