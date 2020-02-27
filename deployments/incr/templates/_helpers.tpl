{{- define "kar.sidecar" -}}
- name: kar-sidecar
  image: {{ .Values.kar.imageRegistry }}/{{ .Values.kar.imageName }}:{{ .Values.kar.imageTag }}
  command: ["/kar/kar", "-app", "$KAR_MANAGED_APP_NAME", "-service", "$KAR_MANAGED_SERVICE_NAME", "/usr/bin/tail", "-f", "/dev/null"]
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
  - name: KAR_MANAGED_APP_NAME
    value: {{ .App }}
  - name: KAR_MANAGED_SERVICE_NAME
    value: {{ .Service }}
  - name: KAR_MANAGED_SERVICE_PORT
    value: {{ .Port | quote }}
{{- end -}}
