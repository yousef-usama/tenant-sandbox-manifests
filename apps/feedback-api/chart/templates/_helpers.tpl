{{- define "feedback-api.name" -}}
feedback-api
{{- end -}}

{{- define "feedback-api.labels" -}}
app.kubernetes.io/name: {{ include "feedback-api.name" . }}
app.kubernetes.io/instance: {{ .Values.tenantName }}
app.kubernetes.io/part-of: tenant-sandbox
tenant: {{ .Values.tenantName }}
{{- end -}}

