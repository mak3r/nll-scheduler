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
# Use custom_build with podman directly — Podman's Docker-compatible API
# does not reliably support the push endpoint used by docker_build.

# Go services: rebuild on .go file changes (fast compile ~5s)
custom_build(
    'nll-scheduler/team-service',
    'podman build --platform linux/arm64 -t $EXPECTED_REF -f team-service/Dockerfile team-service && podman push $EXPECTED_REF',
    ['team-service/cmd/', 'team-service/internal/', 'team-service/go.mod', 'team-service/go.sum'],
    skips_local_docker=True,
)

custom_build(
    'nll-scheduler/field-service',
    'podman build --platform linux/arm64 -t $EXPECTED_REF -f field-service/Dockerfile field-service && podman push $EXPECTED_REF',
    ['field-service/cmd/', 'field-service/internal/', 'field-service/go.mod', 'field-service/go.sum'],
    skips_local_docker=True,
)

custom_build(
    'nll-scheduler/schedule-service',
    'podman build --platform linux/arm64 -t $EXPECTED_REF -f schedule-service/Dockerfile schedule-service && podman push $EXPECTED_REF',
    ['schedule-service/cmd/', 'schedule-service/internal/', 'schedule-service/go.mod', 'schedule-service/go.sum'],
    skips_local_docker=True,
)

# Python service: sync .py files directly into running container
# uvicorn --reload picks up changes without restart
custom_build(
    'nll-scheduler/scheduler-engine',
    'podman build --platform linux/arm64 -t $EXPECTED_REF -f scheduler-engine/Dockerfile scheduler-engine && podman push $EXPECTED_REF',
    ['scheduler-engine/app/', 'scheduler-engine/requirements.txt'],
    skips_local_docker=True,
    live_update=[
        sync('scheduler-engine/app', '/app/app'),
    ],
)

# Frontend: dev stage with Vite HMR
# Source files synced directly; HMR handles hot reload
custom_build(
    'nll-scheduler/frontend',
    'podman build --platform linux/arm64 --target dev -t $EXPECTED_REF -f frontend/Dockerfile frontend && podman push $EXPECTED_REF',
    [
        'frontend/src/',
        'frontend/public/',
        'frontend/index.html',
        'frontend/package.json',
        'frontend/package-lock.json',
        'frontend/vite.config.ts',
        'frontend/tsconfig.json',
        'frontend/tsconfig.node.json',
    ],
    skips_local_docker=True,
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
