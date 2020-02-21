// A non-HA kafka cluster suitable for dev usage.

let solsa = require('solsa')

module.exports = function kafka (values) {
  let zk = new solsa.apps.v1.StatefulSet({
    metadata: { name: `${values.prefix}-zookeeper`, labels: { app: 'kar-runtime' } },
    spec: {
      serviceName: `${values.prefix}-zookeeper`,
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
  let zks = zk.getService()

  return new solsa.Bundle({ zk, zks })
}
