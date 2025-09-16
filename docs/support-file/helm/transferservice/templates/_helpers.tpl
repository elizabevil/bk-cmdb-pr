{{/*
Expand the name of the chart.
*/}}
{{- define "transferservice.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "bk-cmdb.fullname" -}}
{{- $name := default "bk-cmdb" .Values.nameOverride -}}
{{- printf "%s" $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "transferservice.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{ define "transferservice.labels" -}}
helm.sh/chart: {{ include "transferservice.chart" . }}
{{ include "transferservice.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "transferservice.selectorLabels" -}}
app.kubernetes.io/name: {{ include "transferservice.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "bk-cmdb.transferservice" -}}
  {{- printf "%s-transferservice" (include "bk-cmdb.fullname" .) -}}
{{- end -}}

{{- define "cmdb.imagePullSecrets" -}}
{{- if .Values.image.pullSecretName }}
imagePullSecrets:
- name: {{ .Values.image.pullSecretName }}
{{- end }}
{{- end -}}

