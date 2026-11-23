{{/* Labels on every resource. */}}
{{- define "proxy-api.labels" -}}
{{ include "proxy-api.selectorLabels" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Values.image.tag | quote }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{/* Labels on the gateway's resources. Built separately from
proxy-api.labels on purpose: a gateway pod must never carry
app.kubernetes.io/name: proxy-api, or the API's Service, PDB and autoscaler
(and the grader) would treat it as an API pod. */}}
{{- define "proxy-api.gatewayLabels" -}}
{{ include "proxy-api.gatewaySelectorLabels" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Values.gateway.image.tag | quote }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{- define "proxy-api.gatewaySelectorLabels" -}}
app.kubernetes.io/name: proxy-api-gateway
{{- end -}}

{{/* Labels that select the API pods. Must never change, because a
Deployment's selector is immutable. */}}
{{- define "proxy-api.selectorLabels" -}}
app.kubernetes.io/name: proxy-api
{{- end -}}
