# Storybook binary fixtures

`asset-story/content`, `asset-second/content` and their `thumbnail` siblings are byte copies of the synthetic
`tests/fixtures/media/viewer-blue-violet.png`. Storybook maps this directory to
`/api/v1/assets` only for the production people/group-review stories' content/thumbnail requests.
The query cache supplies labelled synthetic contracts; there is no real API,
authentication, recognition, or curation here. Vite production does not serve this
fixture directory. These files are not model quality evidence.
