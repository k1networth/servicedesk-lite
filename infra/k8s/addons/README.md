# Addons for local Kubernetes demo

Для Ingress и HPA в kind обычно нужны два аддона:

- ingress-nginx (controller)
- metrics-server (для HPA по CPU)

## Установка

Самый простой путь (см. Makefile):

```bash
make k8s-addons
```

Ручной запуск (интернет нужен):

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.3/deploy/static/provider/kind/deploy.yaml
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

kubectl -n ingress-nginx wait --for=condition=Ready pod -l app.kubernetes.io/component=controller --timeout=180s
kubectl -n kube-system wait --for=condition=Ready pod -l k8s-app=metrics-server --timeout=180s
```

## Проверка

```bash
kubectl -n ingress-nginx get pods
kubectl -n kube-system get pods | grep metrics-server
kubectl get hpa -n servicedesk
```
