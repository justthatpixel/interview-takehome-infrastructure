{{/* Labels on every resource. */}}
{{- define "proxy-api.labels" -}}
{{ include "proxy-api.selectorLabels" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Values.image.tag | quote }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{/* Labels that select the API pods. Must never change, because a
Deployment's selector is immutable. */}}
{{- define "proxy-api.selectorLabels" -}}
app.kubernetes.io/name: proxy-api
{{- end -}}
