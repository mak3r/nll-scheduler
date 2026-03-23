# NLL Scheduler — Tiltfile
# Run: tilt up
# Requires: kubectl pointing to dev cluster, Tilt installed

# Allow the local dev context (prevents Tilt's production cluster warning)
allow_k8s_contexts('local-dev')

# Push images to ghcr.io so any cluster can pull them
default_registry('ghcr.io/mak3r/nll-scheduler')

# Dev namespace
k8s_yaml('k8s/namespace.yaml')

# --- Image Builds ---

# Go services: rebuild on .go file changes (fast compile ~5s)
docker_build(
    'nll-scheduler/team-service',
    context='team-service',
    dockerfile='team-service/Dockerfile',
    only=['cmd/', 'internal/', 'go.mod', 'go.sum'],
)

docker_build(
    'nll-scheduler/field-service',
    context='field-service',
    dockerfile='field-service/Dockerfile',
    only=['cmd/', 'internal/', 'go.mod', 'go.sum'],
)

docker_build(
    'nll-scheduler/schedule-service',
    context='schedule-service',
    dockerfile='schedule-service/Dockerfile',
    only=['cmd/', 'internal/', 'go.mod', 'go.sum'],
)

# Python service: sync .py files directly into running container
# uvicorn --reload picks up changes without restart
docker_build(
    'nll-scheduler/scheduler-engine',
    context='scheduler-engine',
    dockerfile='scheduler-engine/Dockerfile',
    only=['app/', 'requirements.txt'],
    live_update=[
        sync('scheduler-engine/app', '/app/app'),
    ],
)

# Frontend: dev stage with Vite HMR
# Source files synced directly; HMR handles hot reload
docker_build(
    'nll-scheduler/frontend',
    context='frontend',
    dockerfile='frontend/Dockerfile',
    target='dev',
    only=[
        'src/',
        'public/',
        'index.html',
        'package.json',
        'package-lock.json',
        'vite.config.ts',
        'tsconfig.json',
        'tsconfig.node.json',
    ],
    live_update=[
        sync('frontend/src', '/app/src'),
        sync('frontend/public', '/app/public'),
        sync('frontend/index.html', '/app/index.html'),
    ],
)

# --- K8s Resources ---

# Ingress
k8s_yaml('k8s/ingress.yaml')

# team-service
k8s_yaml([
    'k8s/team-service/pvc.yaml',
    'k8s/team-service/secret.yaml',
    'k8s/team-service/configmap.yaml',
    'k8s/team-service/deployment.yaml',
    'k8s/team-service/service.yaml',
])
k8s_resource(
    'team-service',
    port_forwards=['8081:8080'],
    labels=['backend'],
)

# field-service
k8s_yaml([
    'k8s/field-service/pvc.yaml',
    'k8s/field-service/secret.yaml',
    'k8s/field-service/configmap.yaml',
    'k8s/field-service/deployment.yaml',
    'k8s/field-service/service.yaml',
])
k8s_resource(
    'field-service',
    port_forwards=['8082:8080'],
    labels=['backend'],
)

# schedule-service
k8s_yaml([
    'k8s/schedule-service/pvc.yaml',
    'k8s/schedule-service/secret.yaml',
    'k8s/schedule-service/configmap.yaml',
    'k8s/schedule-service/deployment.yaml',
    'k8s/schedule-service/service.yaml',
])
k8s_resource(
    'schedule-service',
    port_forwards=['8083:8080'],
    labels=['backend'],
)

# scheduler-engine (ClusterIP only — internal service)
# Port-forward available in dev for direct testing
k8s_yaml([
    'k8s/scheduler-engine/deployment.yaml',
    'k8s/scheduler-engine/service.yaml',
])
k8s_resource(
    'scheduler-engine',
    port_forwards=['8084:8080'],
    labels=['backend'],
)

# frontend
k8s_yaml([
    'k8s/frontend/deployment.yaml',
    'k8s/frontend/service.yaml',
])
k8s_resource(
    'frontend',
    port_forwards=['3000:3000'],
    labels=['frontend'],
)
