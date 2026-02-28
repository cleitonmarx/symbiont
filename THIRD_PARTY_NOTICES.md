# Third-Party Notices

This repository vendors browser-side JavaScript assets under [`introspection/mermaid/assets`](./introspection/mermaid/assets) for the Mermaid introspection HTTP handler.

These vendored files are distributed under their respective upstream licenses. This notice file records the upstream package names, license families, and the bundled third-party notices that were discoverable from the vendored distributions at the time these files were added.

## Vendored browser assets

### Mermaid
- Package: `mermaid`
- Vendored files: `introspection/mermaid/assets/mermaid.esm.min.mjs` and `introspection/mermaid/assets/chunks/mermaid.esm.min/*`
- Vendored version: `11.12.3`
- Original CDN pin used when these assets were added: `mermaid@11`
- License: `MIT`
- Upstream:
  - https://github.com/mermaid-js/mermaid
  - https://www.npmjs.com/package/mermaid
- License text: [`LICENSES/MIT.txt`](./LICENSES/MIT.txt)

### Mermaid ELK layout loader
- Package: `@mermaid-js/layout-elk`
- Vendored files: `introspection/mermaid/assets/mermaid-layout-elk.esm.min.mjs` and `introspection/mermaid/assets/chunks/mermaid-layout-elk.esm.min/*`
- Original CDN pin used when these assets were added: `@mermaid-js/layout-elk@0`
- License: `MIT`
- Upstream:
  - https://github.com/mermaid-js/mermaid
  - https://www.npmjs.com/package/@mermaid-js/layout-elk
- License text: [`LICENSES/MIT.txt`](./LICENSES/MIT.txt)
- Note: the vendored loader file does not expose a more specific version string than the original CDN pin. When updating this asset, record the exact upstream package version alongside the update.

### Panzoom
- Package: `@panzoom/panzoom`
- Vendored file: `introspection/mermaid/assets/panzoom.min.js`
- Vendored version: `4.6.1`
- License: `MIT`
- Upstream:
  - https://github.com/timmywil/panzoom
  - https://www.npmjs.com/package/@panzoom/panzoom
- License text: [`LICENSES/MIT.txt`](./LICENSES/MIT.txt)

## Bundled notices preserved inside the Mermaid distribution

The vendored Mermaid distribution includes preserved third-party notices in its chunk files. Based on those preserved notices, the following bundled third-party components should also be treated as carrying attribution requirements:

### DOMPurify
- Package: `dompurify`
- Observed bundled version: `3.2.6`
- Observed notice: `Released under the Apache license 2.0 and Mozilla Public License 2.0`
- Upstream:
  - https://github.com/cure53/DOMPurify
  - https://www.npmjs.com/package/dompurify
- License texts:
  - [`LICENSES/Apache-2.0.txt`](./LICENSES/Apache-2.0.txt)
  - [`LICENSES/MPL-2.0.txt`](./LICENSES/MPL-2.0.txt)

### js-yaml
- Package: `js-yaml`
- Observed bundled version: `4.1.0`
- License: `MIT`
- Upstream:
  - https://github.com/nodeca/js-yaml
  - https://www.npmjs.com/package/js-yaml
- License text: [`LICENSES/MIT.txt`](./LICENSES/MIT.txt)

## Maintenance note

If the vendored browser assets are updated, review this file and the `LICENSES/` directory at the same time. In particular:
- update exact upstream versions where they are known
- re-check whether the vendored bundles preserve notices for additional third-party components
- keep the corresponding license texts in `LICENSES/`
