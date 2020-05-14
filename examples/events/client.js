const fetch = require('fetch-retry')(require('node-fetch'), { retries: 10 })
const v1 = require('cloudevents-sdk/v1')

const base = `http://127.0.0.1:${process.env.KAR_RUNTIME_PORT}/kar/v1`

// publish an event to a topic
async function publish (topic, event) {
  const res = await fetch(`${base}/event/${topic}/publish`, { method: 'POST', body: JSON.stringify(event) })
  return res.text()
}

// main function
async function main () {
  while (true) {
    // construct event
    const event = v1.event()
      .type('test.event')
      .source('test.source')
      .data(Date.now())

    // publish event
    console.log('publish:', await publish('test-topic', event))

    // sleep 1s
    await new Promise(resolve => setTimeout(resolve, 1000))
  }
}

main()
