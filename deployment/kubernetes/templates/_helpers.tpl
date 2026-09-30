{{- define "mip.keycloakAuthUrl" -}}
{{- $protocol := default "https" .Values.keycloak.protocol -}}
{{- $host := default "" .Values.keycloak.host -}}
{{- if $host -}}
{{- printf "%s://%s/auth/" $protocol $host -}}
{{- end -}}
{{- end -}}

{{- define "mip.storageClass" -}}
{{- ternary .Values.cluster.storageClasses.managed .Values.cluster.storageClasses.local .Values.cluster.managed -}}
{{- end -}}
