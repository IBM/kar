const { sync } = require('./kar')

async function main () {
  let x = 0
  for (let i = 0; i < 5000; i++) {
    x = await sync('myService', 'incr', x)
    console.log(i, '->', x)
  }
  console.log('=>', x)
}

main()
