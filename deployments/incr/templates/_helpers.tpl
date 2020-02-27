{{- define "kar.sidecar" -}}
- name: kar
  image: {{ .Values.kar.imageRegistry }}/{{ .Values.kar.imageName }}:{{ .Values.kar.imageTag }}
  command: ["/kar/kar", "-verbose", "-app", "{{ .App }}", "-service", "{{ .Service }}", "-port", "{{ .ServicePort }}", "-listen", "{{ .ListenPort }}" ]
  imagePullPolicy: Always
  env:
  - name: KAFKA_BROKERS
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: kafka_brokers
  - name: KAFKA_USER
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: kafka_user
  - name: KAFKA_PASSWORD
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: kafka_password
  - name: REDIS_ADDRESS
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: redis_address
  - name: REDIS_PASSWORD
    valueFrom:
      configMapKeyRef:
        name: {{ .Values.kar.runtimeConfigName }}
        key: redis_password
{{- end -}}
