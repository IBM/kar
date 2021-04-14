{{/* vim: set filetype=mustache: */}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kar.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "kar.labels" -}}
helm.sh/chart: {{ include "kar.chart" . }}
{{ include "kar.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "kar.selectorLabels" -}}
app.kubernetes.io/name: kar
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* host name for server.N in zookeeper cluster */}}
{{- define "kar.zookeeper_host_zero" -}}
kar-zookeeper-0.kar-zookeeper.{{ $.Release.Namespace }}.svc.{{ .Values.global.clusterSubDomain }}
{{- end -}}
{{- define "kar.zookeeper_host_one" -}}
kar-zookeeper-1.kar-zookeeper.{{ $.Release.Namespace }}.svc.{{ .Values.global.clusterSubDomain }}
{{- end -}}
{{- define "kar.zookeeper_host_two" -}}
kar-zookeeper-1.kar-zookeeper.{{ $.Release.Namespace }}.svc.{{ .Values.global.clusterSubDomain }}
{{- end -}}

{{/* host name for server.N in kafka cluster */}}
{{- define "kar.kafka_host_zero" -}}
kar-kafka-0.kar-kafka.{{ $.Release.Namespace }}.svc.{{ .Values.global.clusterSubDomain }}
{{- end -}}
{{- define "kar.kafka_host_one" -}}
kar-kafka-1.kar-kafka.{{ $.Release.Namespace }}.svc.{{ .Values.global.clusterSubDomain }}
{{- end -}}
{{- define "kar.kafka_host_two" -}}
kar-kafka-2.kar-kafka.{{ $.Release.Namespace }}.svc.{{ .Values.global.clusterSubDomain }}
{{- end -}}
