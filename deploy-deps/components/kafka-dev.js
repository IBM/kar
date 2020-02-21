// A non-HA kafka cluster suitable for dev usage.

let solsa = require('solsa')

module.exports = function kafka (values) {
  let zkName = `${values.prefix}-zookeeper`
  let zk = new solsa.apps.v1.StatefulSet({
    metadata: { name: zkName, labels: { app: 'kar-runtime' } },
    spec: {
      serviceName: zkName,
      selector: { matchLabels: { 'solsa.ibm.com/pod': `${values.prefix}-zk` } },
      replicas: 1,
      template: {
        spec: {
          containers: [
            {
              name: 'zk',
              image: `${values.zk.imageRegistry}/${values.zk.imageName}:${values.zk.imageTag}`,
              ports: [
                { name: 'zookeeper', containerPort: values.zk.port },
                { name: 'server', containerPort: values.zk.serverPort },
                { name: 'leader-election', containerPort: values.zk.leaderElectionPort }
              ]
            }
          ]
        }
      }
    }
  })
  if (values.zk.enableProbes) {
    zk.spec.template.spec.containers[0].livenessProbe = { tcpSocket: { port: values.zk.port } }
    zk.spec.template.spec.containers[0].readinessProbe = { exec: { command: [ '/bin/bash', '-c', `echo ruok | nc -w 1 localhost ${values.zk.port} | grep imok` ] } }
  }
  zk.propogateLabels()
  let zksvc = zk.getService()
  zksvc.spec.clusterIP = 'None'

  let kafka = new solsa.apps.v1.StatefulSet({
    metadata: { name: `${values.prefix}-kafka`, labels: { app: 'kar-runtime' } },
    spec: {
      serviceName: `${values.prefix}-kafka`,
      selector: { matchLabels: { 'solsa.ibm.com/pod': `${values.prefix}-kafka` } },
      replicas: 1,
      template: {
        spec: {
          containers: [
            {
              name: 'kafka',
              image: `${values.kafka.imageRegistry}/${values.kafka.imageName}:${values.kafka.imageTag}`,
              ports: [
                { name: 'kafka', containerPort: values.kafka.port }
              ],
              env: [
                { name: 'HOSTNAME_COMMAND', value: 'hostname -f' },
                { name: 'KAFKA_ADVERTISED_PORT', value: `${values.kafka.port}` },
                { name: 'KAFKA_PORT', value: `${values.kafka.port}` },
                { name: 'KAFKA_LISTENER_SECURITY_PROTOCOL_MAP', value: 'EXTERNAL:PLAINTEXT' },
                { name: 'KAFKA_LISTENERS', value: `EXTERNAL://:${values.kafka.port}` },
                { name: 'KAFKA_ADVERTISED_LISTENERS', value: `EXTERNAL://_{HOSTNAME_COMMAND}:${values.kafka.port}` },
                { name: 'KAFKA_INTER_BROKER_LISTENER_NAME', value: 'EXTERNAL' },
                { name: 'KAFKA_ZOOKEEPER_CONNECT', value: `${zkName}-0.${zkName}:${values.zk.port}` }
              ]
            }
          ]
        }
      }
    }
  })
  if (values.kafka.enableProbes) {
    kafka.spec.template.spec.containers[0].livenessProbe = { tcpSocket: { port: values.kafka.port } }
  }
  kafka.propogateLabels()
  let kafkasvc = kafka.getService()
  kafkasvc.spec.clusterIP = 'None'

  return new solsa.Bundle({ zk, zksvc, kafka, kafkasvc })
}
