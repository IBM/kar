const { actor, sys } = require('kar')

const sleep = (milliseconds) => {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}

const verbose = process.env.VERBOSE

async function testTermination (failure) {
  if (failure) {
    console.log('FAILED; setting non-zero exit code')
    process.exitCode = 1
  } else {
    console.log('SUCCESS')
    process.exitCode = 0
  }

  console.log('Terminating sidecar')
  await sys.shutdown()
}

function prettyPrintHistogram (header, bucketSizeInMS, histogram) {
  const population = histogram.reduce((x, y) => x + y, 0)
  console.log(header)
  console.log('Bucket\tCount\tPercent\tCumulative%')
  let accumPercent = 0
  for (const i in histogram) {
    const value = histogram[i] || 0
    const percent = value / population * 100
    accumPercent += percent
    console.log(`\t<${(parseInt(i) + 1) * bucketSizeInMS}ms\t${value}\t${(percent).toFixed(2)}\t${(accumPercent).toFixed(2)}`)
  }
}

async function summaryReport (division) {
  const summary = { onboarding: 0, home: 0, commuting: 0, working: 0, meeting: 0, coffee: 0, lunch: 0 }
  for (const site of division) {
    const sr = await actor.call(site.proxy, 'siteReport')
    summary.onboarding += sr.onboarding || 0
    summary.home += sr.home || 0
    summary.commuting += sr.commuting || 0
    summary.working += sr.working || 0
    summary.meeting += sr.meeting || 0
    summary.coffee += sr.coffee || 0
    summary.lunch += sr.lunch || 0
  }
  return summary
}

async function main () {
  let failure = false

  const ibm = actor.proxy('Company', 'IBM')
  const researchDivision = [
    { proxy: actor.proxy('Site', 'Yorktown'), params: { workers: 20, thinkms: 2000, steps: 10, days: 2 } },
    { proxy: actor.proxy('Site', 'Cambridge'), params: { workers: 10, thinkms: 1000, steps: 40, days: 1 } },
    { proxy: actor.proxy('Site', 'Almaden'), params: { workers: 15, thinkms: 500, steps: 10, days: 5 } }
  ]

  console.log('Starting simulation:')
  for (const site of researchDivision) {
    console.log(`  ${site.proxy.kar.id}: ${JSON.stringify(site.params)}`)
  }

  for (const site of researchDivision) {
    await actor.call(site.proxy, 'resetDelayStats')
    await actor.reminders.schedule(site.proxy, 'siteReport', { id: 'clientSiteReport', targetTime: new Date(Date.now() + 1000), period: '1s' })
    await actor.call(ibm, 'hire', Object.assign({ site: site.proxy.kar.id }, site.params))
  }

  while (true) {
    await sleep(5000)
    const employees = await actor.call(ibm, 'count')
    console.log(`Num employees is ${employees}`)
    if (employees === 0) {
      const summary = { reminderDelays: [], tellLatencies: [] }
      let bucketSizeInMS
      for (const site of researchDivision) {
        console.log(`Valiadating ${site.proxy.kar.id}`)
        const sr = await actor.call(site.proxy, 'siteReport')
        if (sr.siteEmployees !== 0) {
          console.log(`FAILURE: ${sr.siteEmployees} stranded employees at ${site.proxy.kar.id}`)
          failure = true
        }
        const count = sr.reminderDelays.reduce((x, y) => x + y, 0)
        const expectedSteps = site.params.workers * (site.params.steps * site.params.days + 1)
        if (count !== expectedSteps) {
          console.log(`FAILURE: At ${site.proxy.kar.id} expected ${expectedSteps} steps, but actual value is ${count}`)
          failure = true
        }
        bucketSizeInMS = sr.bucketSizeInMS
        for (const idx in sr.reminderDelays) {
          summary.reminderDelays[idx] = (sr.reminderDelays[idx] || 0) + (summary.reminderDelays[idx] || 0)
        }
        for (const idx in sr.workerUpdateLatency) {
          summary.tellLatencies[idx] = (sr.workerUpdateLatency[idx] || 0) + (summary.tellLatencies[idx] || 0)
        }
      }
      prettyPrintHistogram('Reminder Delays', bucketSizeInMS, summary.reminderDelays)
      prettyPrintHistogram('Tell Latency', bucketSizeInMS, summary.tellLatencies)

      break
    } else if (verbose) {
      const summary = await summaryReport(researchDivision)
      console.log(summary)
    }
  }

  testTermination(failure)
}

main()
