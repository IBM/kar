const { events } = require('kar')
const v1 = require('cloudevents-sdk/v1')

// main function
async function main () {
  while (true) {
    // construct event
    const event = v1.event()
      .type('test.event')
      .source('test.source')
      .data(Date.now())

    // publish event
    console.log('publish:', await events.publish('test-topic', event))

    // sleep 1s
    await new Promise(resolve => setTimeout(resolve, 1000))
  }
}

main()
