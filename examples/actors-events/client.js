const { events } = require('kar')
const { CloudEvent } = require('cloudevents')

// main function
async function main () {
  while (true) {
    // construct event
    const event = new CloudEvent({
      type: 'test.event',
      source: 'test.source',
      data: Date.now()
    })

    // publish event
    console.log('publish:', await events.publish('test-topic', event))

    // sleep 1s
    await new Promise(resolve => setTimeout(resolve, 1000))
  }
}

main()
