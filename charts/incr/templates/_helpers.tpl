{{- define "kar.sidecar" -}}
- name: kar
  image: {{ .Values.kar.imageRegistry }}/{{ .Values.kar.imageName }}:{{ .Values.kar.imageTag }}
  command: ["/kar/kar", "-v", "4", "-app", "{{ .App }}", "-service", "{{ .Service }}", "-send", "{{ .ServicePort }}", "-recv", "{{ .RuntimePort }}" ]
  imagePullPolicy: Always
  env:
  - name: KAFKA_BROKERS
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: kafka_brokers
  - name: KAFKA_USERNAME
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: kafka_username
  - name: KAFKA_PASSWORD
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: kafka_password
  - name: REDIS_HOST
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: redis_host
  - name: REDIS_PORT
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: redis_port
  - name: REDIS_PASSWORD
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: redis_password
{{- end -}}
