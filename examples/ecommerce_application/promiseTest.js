const express = require('express')
const { actor, sys } = require('kar-sdk')

class Moo {
  async activate() {
    console.log('Here')
  }

  async makeRequest(i) {
      return new Promise((resolve) => {
        setTimeout(() => resolve({ 'status': 'done', val: i}), 2000);
      });
  }

  async  process(arrayOfPromises) {
      let responses = []
      console.time(`process`);
      for (let a in arrayOfPromises) { responses.push(await arrayOfPromises[a])}
      // let responses = await Promise.all(arrayOfPromises);
      console.timeEnd(`process`);
      for (let r in responses) {console.log(responses[r].val)}
      return;
  }
  async handler() {
      let arrayOfPromises = [
          this.makeRequest(1),
          this.makeRequest(2),
          this.makeRequest(3),
          this.makeRequest(4),
          this.makeRequest(5),
      ];
      
      await this.process(arrayOfPromises);
      console.log(`processing is complete`);
  }
}

async function main() {
  const c = actor.proxy('Moo', 123)
  await actor.call(c, 'handler')
}

const app = express()
app.use(sys.actorRuntime({ Moo }))
sys.h2c(app).listen(process.env.KAR_APP_PORT, process.env.KAR_APP_HOST || '127.0.0.1')

main()
// c.handler();