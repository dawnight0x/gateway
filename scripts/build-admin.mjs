import { compile } from '@vue/compiler-dom';
import { copyFile, readFile, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const sourcePath = resolve(root, 'web/admin/index.template.html');
const outputPath = resolve(root, 'web/admin/index.html');
const renderPath = resolve(root, 'web/admin/assets/render.js');
const runtimePath = resolve(root, 'web/admin/assets/vendor/vue.runtime.global.prod.js');
const source = await readFile(sourcePath, 'utf8');

const appMarker = '<div id="app" v-cloak>';
const appStart = source.indexOf(appMarker);
const scriptsStart = source.indexOf('  <script src="/admin/assets/vendor/vue.runtime.global.prod.js"></script>');
if (appStart < 0 || scriptsStart < 0) {
  throw new Error('admin template markers were not found');
}
const appEnd = source.lastIndexOf('  </div>', scriptsStart);
if (appEnd < appStart) {
  throw new Error('admin template root closing tag was not found');
}
const template = source.slice(appStart + appMarker.length, appEnd);
const result = compile(template, {
  mode: 'module',
  prefixIdentifiers: true,
  hoistStatic: true,
  cacheHandlers: true,
  filename: 'web/admin/index.template.html',
});

let render = result.code.replace(
  /import\s*\{([\s\S]*?)\}\s*from\s*["']vue["'];?\s*/,
  (_, bindings) => `const { ${bindings.replaceAll(' as ', ': ')} } = Vue;\n`,
);
render = render.replace('export function render', 'window.gatewayRender = function render');
if (!render.includes('window.gatewayRender')) {
  throw new Error('compiled render function could not be converted to the Vue global runtime');
}
await writeFile(renderPath, `${render}\n`, 'utf8');
await copyFile(resolve(root, 'node_modules/vue/dist/vue.runtime.global.prod.js'), runtimePath);

const headEnd = source.indexOf('<body>') + '<body>'.length;
const index = `${source.slice(0, headEnd)}
  <div id="app" v-cloak></div>
  <script src="/admin/assets/vendor/vue.runtime.global.prod.js"></script>
  <script src="/admin/assets/i18n.js"></script>
  <script src="/admin/assets/render.js"></script>
  <script src="/admin/assets/app.js"></script>
</body>
</html>
`;
await writeFile(outputPath, index, 'utf8');
