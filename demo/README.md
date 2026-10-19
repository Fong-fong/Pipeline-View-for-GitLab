## Demo without a real GitLab

The screenshots above come from a small mock API with made-up projects, so no real data is involved. To try it yourself:

```sh
go run ./demo/mockgitlab &        # fake GitLab on :8080
go run . --host=http://localhost:8080 --group-id=acme --token=demo
```

To regenerate the screenshots:

```sh
go run . --host=http://localhost:8080 --group-id=acme --token=demo --once \
  | python3 demo/ansi2svg.py "pipeline --group-id=acme" > docs/demo.svg
```
