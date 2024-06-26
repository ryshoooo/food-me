DO
$do$
BEGIN
   IF EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE  rolname = '{{ .preferred_username }}') THEN

      RAISE NOTICE 'Role "{{ .preferred_username }}" already exists. Skipping.';
   ELSE
      CREATE USER {{ .preferred_username }};
   END IF;
END
$do$;

{{- $is_superuser := false -}}
{{ range $group := .groups }}{{ if eq $group "pgadmin" }}{{ $is_superuser = true }}{{ end }}{{ end }}
{{- if $is_superuser -}}
ALTER USER {{ .preferred_username }} WITH SUPERUSER;
{{- else -}}
ALTER USER {{ .preferred_username }} WITH NOSUPERUSER;
{{- end -}}

