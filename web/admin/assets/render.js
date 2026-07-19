const {  createElementVNode: _createElementVNode, toDisplayString: _toDisplayString, renderList: _renderList, Fragment: _Fragment, openBlock: _openBlock, createElementBlock: _createElementBlock, withModifiers: _withModifiers, normalizeClass: _normalizeClass, createTextVNode: _createTextVNode, createCommentVNode: _createCommentVNode, vShow: _vShow, withDirectives: _withDirectives, vModelText: _vModelText, vModelSelect: _vModelSelect, vModelCheckbox: _vModelCheckbox, withKeys: _withKeys, createStaticVNode: _createStaticVNode  } = Vue;
const _hoisted_1 = { class: "sidebar" }
const _hoisted_2 = { class: "brand" }
const _hoisted_3 = { class: "nav" }
const _hoisted_4 = ["href", "onClick"]
const _hoisted_5 = {
  class: "nav-icon",
  "aria-hidden": "true"
}
const _hoisted_6 = { class: "side-status" }
const _hoisted_7 = { class: "shell" }
const _hoisted_8 = { class: "topbar" }
const _hoisted_9 = { class: "headline" }
const _hoisted_10 = { class: "top-actions" }
const _hoisted_11 = ["aria-label"]
const _hoisted_12 = ["disabled"]
const _hoisted_13 = {
  key: 0,
  class: "refresh-error",
  role: "alert"
}
const _hoisted_14 = { class: "section" }
const _hoisted_15 = { class: "hero-panel" }
const _hoisted_16 = { class: "eyebrow" }
const _hoisted_17 = { class: "metric-grid" }
const _hoisted_18 = { class: "split" }
const _hoisted_19 = { class: "panel" }
const _hoisted_20 = { class: "panel-title" }
const _hoisted_21 = { class: "panel-actions" }
const _hoisted_22 = { class: "compact-list" }
const _hoisted_23 = {
  key: 0,
  class: "empty-state"
}
const _hoisted_24 = { class: "panel" }
const _hoisted_25 = { class: "panel-title traffic-heading" }
const _hoisted_26 = { class: "log-list traffic-list" }
const _hoisted_27 = { class: "sr-only" }
const _hoisted_28 = { class: "traffic-primary" }
const _hoisted_29 = { class: "traffic-status" }
const _hoisted_30 = { class: "traffic-protocol" }
const _hoisted_31 = { class: "traffic-latency" }
const _hoisted_32 = ["datetime"]
const _hoisted_33 = { class: "traffic-secondary" }
const _hoisted_34 = {
  key: 0,
  class: "traffic-provider"
}
const _hoisted_35 = {
  key: 1,
  class: "traffic-provider"
}
const _hoisted_36 = {
  key: 2,
  class: "traffic-provider"
}
const _hoisted_37 = {
  key: 3,
  class: "traffic-error"
}
const _hoisted_38 = {
  key: 0,
  class: "empty-state compact-empty"
}
const _hoisted_39 = { class: "section" }
const _hoisted_40 = { class: "panel" }
const _hoisted_41 = { class: "panel-title section-actions" }
const _hoisted_42 = { class: "panel-actions" }
const _hoisted_43 = ["disabled"]
const _hoisted_44 = { class: "provider-summary-grid" }
const _hoisted_45 = { class: "summary-card" }
const _hoisted_46 = { class: "summary-card" }
const _hoisted_47 = { class: "summary-card" }
const _hoisted_48 = {
  key: 0,
  class: "form-error"
}
const _hoisted_49 = ["disabled", "placeholder"]
const _hoisted_50 = ["placeholder"]
const _hoisted_51 = ["value"]
const _hoisted_52 = ["placeholder"]
const _hoisted_53 = ["placeholder"]
const _hoisted_54 = ["placeholder"]
const _hoisted_55 = { class: "model-select-field" }
const _hoisted_56 = { value: "" }
const _hoisted_57 = ["value"]
const _hoisted_58 = { class: "form-section" }
const _hoisted_59 = ["placeholder"]
const _hoisted_60 = ["placeholder"]
const _hoisted_61 = ["placeholder"]
const _hoisted_62 = { class: "advanced-field" }
const _hoisted_63 = ["placeholder"]
const _hoisted_64 = { class: "quick-fill" }
const _hoisted_65 = { class: "form-actions" }
const _hoisted_66 = { type: "submit" }
const _hoisted_67 = { class: "provider-board" }
const _hoisted_68 = { class: "provider-card-main" }
const _hoisted_69 = { class: "provider-title-row" }
const _hoisted_70 = { class: "pill" }
const _hoisted_71 = { class: "provider-endpoint" }
const _hoisted_72 = { class: "provider-actions" }
const _hoisted_73 = ["onClick"]
const _hoisted_74 = ["onClick"]
const _hoisted_75 = ["onClick"]
const _hoisted_76 = ["onClick"]
const _hoisted_77 = { class: "inline-form-heading" }
const _hoisted_78 = {
  key: 0,
  class: "form-error"
}
const _hoisted_79 = ["placeholder"]
const _hoisted_80 = ["placeholder"]
const _hoisted_81 = ["value"]
const _hoisted_82 = ["placeholder"]
const _hoisted_83 = ["placeholder"]
const _hoisted_84 = ["placeholder"]
const _hoisted_85 = { class: "model-select-field" }
const _hoisted_86 = { value: "" }
const _hoisted_87 = ["value"]
const _hoisted_88 = { class: "advanced-field" }
const _hoisted_89 = ["placeholder"]
const _hoisted_90 = { class: "quick-fill" }
const _hoisted_91 = { class: "form-actions" }
const _hoisted_92 = { type: "submit" }
const _hoisted_93 = { class: "provider-meta-grid" }
const _hoisted_94 = { class: "model-inventory" }
const _hoisted_95 = { class: "model-inventory-head" }
const _hoisted_96 = { class: "model-inventory-title" }
const _hoisted_97 = { class: "pill" }
const _hoisted_98 = { key: 0 }
const _hoisted_99 = { key: 1 }
const _hoisted_100 = ["disabled", "onClick"]
const _hoisted_101 = {
  key: 0,
  class: "model-chip-list"
}
const _hoisted_102 = {
  key: 0,
  class: "model-chip-more"
}
const _hoisted_103 = {
  key: 1,
  class: "test-error"
}
const _hoisted_104 = { key: 2 }
const _hoisted_105 = { class: "inline-form-heading" }
const _hoisted_106 = {
  key: 0,
  class: "form-error"
}
const _hoisted_107 = ["value"]
const _hoisted_108 = ["placeholder"]
const _hoisted_109 = ["placeholder"]
const _hoisted_110 = ["placeholder"]
const _hoisted_111 = { class: "model-select-field" }
const _hoisted_112 = { value: "" }
const _hoisted_113 = ["value"]
const _hoisted_114 = { class: "check" }
const _hoisted_115 = { class: "form-actions" }
const _hoisted_116 = { type: "submit" }
const _hoisted_117 = {
  key: 2,
  class: "key-stack"
}
const _hoisted_118 = { class: "key-card-main" }
const _hoisted_119 = { class: "key-title-row" }
const _hoisted_120 = { class: "key-name" }
const _hoisted_121 = { class: "pill" }
const _hoisted_122 = {
  key: 0,
  class: "pill ok"
}
const _hoisted_123 = { class: "key-hint" }
const _hoisted_124 = { class: "inline-form-heading" }
const _hoisted_125 = ["title"]
const _hoisted_126 = {
  key: 0,
  class: "form-error"
}
const _hoisted_127 = ["value"]
const _hoisted_128 = ["placeholder"]
const _hoisted_129 = ["placeholder"]
const _hoisted_130 = ["placeholder"]
const _hoisted_131 = { class: "model-select-field" }
const _hoisted_132 = { value: "" }
const _hoisted_133 = ["value"]
const _hoisted_134 = { class: "check" }
const _hoisted_135 = { class: "form-actions" }
const _hoisted_136 = { type: "submit" }
const _hoisted_137 = { class: "key-test-row" }
const _hoisted_138 = ["onClick", "disabled"]
const _hoisted_139 = {
  key: 2,
  class: "test-error"
}
const _hoisted_140 = { class: "actions key-actions" }
const _hoisted_141 = ["onClick"]
const _hoisted_142 = ["onClick"]
const _hoisted_143 = ["onClick"]
const _hoisted_144 = ["onClick"]
const _hoisted_145 = ["onClick"]
const _hoisted_146 = {
  key: 3,
  class: "empty-state compact-empty provider-empty"
}
const _hoisted_147 = ["onClick"]
const _hoisted_148 = {
  key: 0,
  class: "empty-state compact-empty"
}
const _hoisted_149 = { class: "balance-header" }
const _hoisted_150 = ["disabled"]
const _hoisted_151 = {
  key: 1,
  class: "compact-list balance-result-list"
}
const _hoisted_152 = { class: "entity-label" }
const _hoisted_153 = ["title"]
const _hoisted_154 = { class: "table-wrap balance-table-wrap" }
const _hoisted_155 = { class: "entity-label" }
const _hoisted_156 = ["title"]
const _hoisted_157 = { class: "entity-label" }
const _hoisted_158 = ["title"]
const _hoisted_159 = { key: 0 }
const _hoisted_160 = { colspan: "6" }
const _hoisted_161 = { class: "empty-state compact-empty" }
const _hoisted_162 = { class: "section" }
const _hoisted_163 = { class: "panel model-routing-panel" }
const _hoisted_164 = { class: "panel-title section-actions model-routing-heading" }
const _hoisted_165 = { class: "model-route-editor-head" }
const _hoisted_166 = { class: "check" }
const _hoisted_167 = {
  key: 0,
  class: "form-error"
}
const _hoisted_168 = { class: "model-route-identity" }
const _hoisted_169 = ["disabled"]
const _hoisted_170 = { class: "route-model-list" }
const _hoisted_171 = { class: "route-model-main" }
const _hoisted_172 = { class: "route-order" }
const _hoisted_173 = ["onUpdate:modelValue"]
const _hoisted_174 = { class: "priority-field" }
const _hoisted_175 = ["onUpdate:modelValue"]
const _hoisted_176 = { class: "check" }
const _hoisted_177 = ["onUpdate:modelValue"]
const _hoisted_178 = ["aria-label", "title", "onClick"]
const _hoisted_179 = { class: "route-target-list" }
const _hoisted_180 = ["onUpdate:modelValue", "aria-label"]
const _hoisted_181 = {
  value: "",
  disabled: ""
}
const _hoisted_182 = ["value"]
const _hoisted_183 = ["onUpdate:modelValue", "list", "aria-label", "placeholder"]
const _hoisted_184 = ["id"]
const _hoisted_185 = ["value"]
const _hoisted_186 = { class: "check" }
const _hoisted_187 = ["onUpdate:modelValue"]
const _hoisted_188 = ["aria-label", "title", "onClick"]
const _hoisted_189 = ["onClick"]
const _hoisted_190 = { class: "form-actions model-route-actions" }
const _hoisted_191 = { type: "submit" }
const _hoisted_192 = {
  key: 1,
  class: "model-route-list"
}
const _hoisted_193 = { class: "model-route-summary" }
const _hoisted_194 = { class: "provider-title-row" }
const _hoisted_195 = { class: "provider-actions" }
const _hoisted_196 = ["onClick"]
const _hoisted_197 = ["onClick"]
const _hoisted_198 = ["onClick"]
const _hoisted_199 = { class: "route-ladder" }
const _hoisted_200 = { class: "route-order" }
const _hoisted_201 = { class: "route-ladder-model" }
const _hoisted_202 = { class: "route-target-chips" }
const _hoisted_203 = { key: 0 }
const _hoisted_204 = {
  key: 2,
  class: "empty-state compact-empty"
}
const _hoisted_205 = { class: "section" }
const _hoisted_206 = { class: "panel" }
const _hoisted_207 = { class: "flow-hint" }
const _hoisted_208 = ["placeholder"]
const _hoisted_209 = { class: "form-actions" }
const _hoisted_210 = { type: "submit" }
const _hoisted_211 = {
  key: 0,
  class: "flow-hint"
}
const _hoisted_212 = { class: "key-hint" }
const _hoisted_213 = { class: "flow-hint" }
const _hoisted_214 = {
  key: 1,
  class: "empty-state compact-empty"
}
const _hoisted_215 = {
  key: 2,
  class: "table-wrap"
}
const _hoisted_216 = { class: "key-hint" }
const _hoisted_217 = { class: "actions" }
const _hoisted_218 = ["onClick"]
const _hoisted_219 = ["onClick"]
const _hoisted_220 = ["onClick"]
const _hoisted_221 = { class: "section" }
const _hoisted_222 = { class: "panel" }
const _hoisted_223 = { class: "routing-section" }
const _hoisted_224 = { class: "routing-section-heading" }
const _hoisted_225 = { class: "routing-fields" }
const _hoisted_226 = { class: "check" }
const _hoisted_227 = { class: "routing-advanced-toggle" }
const _hoisted_228 = ["aria-expanded"]
const _hoisted_229 = {
  key: 0,
  id: "routing-advanced",
  class: "routing-section routing-advanced-section"
}
const _hoisted_230 = { class: "routing-section-heading" }
const _hoisted_231 = { class: "routing-fields" }
const _hoisted_232 = { class: "routing-risk" }
const _hoisted_233 = { class: "check" }
const _hoisted_234 = { class: "check" }
const _hoisted_235 = { class: "form-actions" }
const _hoisted_236 = { type: "submit" }
const _hoisted_237 = {
  key: 0,
  class: "form-success",
  role: "status"
}
const _hoisted_238 = { class: "panel-title maintenance-actions" }
const _hoisted_239 = { class: "panel-actions" }
const _hoisted_240 = { type: "submit" }
const _hoisted_241 = {
  key: 1,
  class: "form-success",
  role: "status"
}
const _hoisted_242 = { class: "section" }
const _hoisted_243 = { class: "panel" }
const _hoisted_244 = { class: "panel-title section-actions" }
const _hoisted_245 = { class: "flow-hint" }
const _hoisted_246 = { class: "snippet-grid" }
const _hoisted_247 = { class: "snippet-head" }
const _hoisted_248 = ["onClick"]
const _hoisted_249 = { class: "section" }
const _hoisted_250 = { class: "panel" }
const _hoisted_251 = { class: "log-toolbar" }
const _hoisted_252 = ["aria-label", "placeholder"]
const _hoisted_253 = ["aria-label"]
const _hoisted_254 = { value: "" }
const _hoisted_255 = ["aria-label"]
const _hoisted_256 = { value: "" }
const _hoisted_257 = ["value"]
const _hoisted_258 = ["aria-label", "placeholder"]
const _hoisted_259 = ["disabled"]
const _hoisted_260 = { class: "table-wrap" }
const _hoisted_261 = { class: "log-model-cell" }
const _hoisted_262 = { key: 0 }
const _hoisted_263 = { key: 1 }
const _hoisted_264 = { class: "pagination" }
const _hoisted_265 = ["disabled"]
const _hoisted_266 = ["disabled"]

window.gatewayRender = function render(_ctx, _cache) {
  return (_openBlock(), _createElementBlock(_Fragment, null, [
    _createElementVNode("aside", _hoisted_1, [
      _createElementVNode("div", _hoisted_2, [
        _cache[99] || (_cache[99] = _createElementVNode("img", {
          class: "brand-mark",
          src: "/admin/assets/app-icon.svg",
          alt: ""
        }, null, -1 /* CACHED */)),
        _createElementVNode("div", null, [
          _cache[98] || (_cache[98] = _createElementVNode("strong", null, "Local AI Gateway", -1 /* CACHED */)),
          _createElementVNode("small", null, "localhost:" + _toDisplayString(_ctx.port), 1 /* TEXT */)
        ])
      ]),
      _createElementVNode("nav", _hoisted_3, [
        (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.navItems, (item) => {
          return (_openBlock(), _createElementBlock("a", {
            key: item.id,
            href: '#' + item.id,
            class: _normalizeClass({active: _ctx.activeSection === item.id}),
            onClick: _withModifiers($event => (_ctx.setActiveSection(item.id)), ["prevent"])
          }, [
            _createElementVNode("span", _hoisted_5, _toDisplayString(item.icon), 1 /* TEXT */),
            _createElementVNode("span", null, _toDisplayString(_ctx.t(item.label)), 1 /* TEXT */)
          ], 10 /* CLASS, PROPS */, _hoisted_4))
        }), 128 /* KEYED_FRAGMENT */))
      ]),
      _createElementVNode("div", _hoisted_6, [
        _createElementVNode("span", {
          class: _normalizeClass(["status-dot", {muted: !_ctx.dashboard}])
        }, null, 2 /* CLASS */),
        _createElementVNode("div", null, [
          _createElementVNode("strong", null, _toDisplayString(_ctx.dashboard?.service?.status || '-'), 1 /* TEXT */),
          _createElementVNode("small", null, _toDisplayString(_ctx.dashboard?.service?.proxyUrl || `http://localhost:${_ctx.port}`), 1 /* TEXT */)
        ])
      ])
    ]),
    _createElementVNode("main", _hoisted_7, [
      _createElementVNode("header", _hoisted_8, [
        _createElementVNode("div", _hoisted_9, [
          _createElementVNode("h1", null, _toDisplayString(_ctx.t(_ctx.currentTitle)), 1 /* TEXT */)
        ]),
        _createElementVNode("div", _hoisted_10, [
          _createElementVNode("button", {
            class: "ghost",
            type: "button",
            onClick: _cache[0] || (_cache[0] = (...args) => (_ctx.openSetup && _ctx.openSetup(...args)))
          }, _toDisplayString(_ctx.t('action.programmingSetup')), 1 /* TEXT */),
          _createElementVNode("button", {
            class: "ghost theme-toggle",
            type: "button",
            onClick: _cache[1] || (_cache[1] = (...args) => (_ctx.toggleTheme && _ctx.toggleTheme(...args))),
            "aria-label": _ctx.t('action.theme')
          }, [
            _createElementVNode("span", null, _toDisplayString(_ctx.theme === 'dark' ? '☀' : '☾'), 1 /* TEXT */),
            _createTextVNode(" " + _toDisplayString(_ctx.theme === 'dark' ? _ctx.t('action.lightTheme') : _ctx.t('action.darkTheme')), 1 /* TEXT */)
          ], 8 /* PROPS */, _hoisted_11),
          _createElementVNode("button", {
            class: "ghost",
            type: "button",
            onClick: _cache[2] || (_cache[2] = $event => (_ctx.setLang(_ctx.lang === 'zh' ? 'en' : 'zh')))
          }, _toDisplayString(_ctx.lang === 'zh' ? 'English' : '中文'), 1 /* TEXT */),
          (_ctx.adminToken)
            ? (_openBlock(), _createElementBlock("button", {
                key: 0,
                class: "ghost",
                type: "button",
                onClick: _cache[3] || (_cache[3] = (...args) => (_ctx.logoutAdmin && _ctx.logoutAdmin(...args)))
              }, _toDisplayString(_ctx.t('auth.logout')), 1 /* TEXT */))
            : _createCommentVNode("v-if", true),
          _createElementVNode("button", {
            type: "button",
            disabled: _ctx.refreshing,
            onClick: _cache[4] || (_cache[4] = (...args) => (_ctx.refresh && _ctx.refresh(...args)))
          }, _toDisplayString(_ctx.refreshing ? _ctx.t('action.refreshing') : _ctx.t('action.refresh')), 9 /* TEXT, PROPS */, _hoisted_12)
        ])
      ]),
      (_ctx.refreshError)
        ? (_openBlock(), _createElementBlock("div", _hoisted_13, _toDisplayString(_ctx.refreshError), 1 /* TEXT */))
        : _createCommentVNode("v-if", true),
      _withDirectives(_createElementVNode("section", _hoisted_14, [
        _createElementVNode("article", _hoisted_15, [
          _createElementVNode("div", null, [
            _createElementVNode("p", _hoisted_16, _toDisplayString(_ctx.t('dashboard.gateway')), 1 /* TEXT */),
            _createElementVNode("h2", null, _toDisplayString(_ctx.dashboard?.service?.proxyUrl || `http://localhost:${_ctx.port}`), 1 /* TEXT */),
            _createElementVNode("p", null, _toDisplayString(_ctx.t('dashboard.gatewayCopy')), 1 /* TEXT */)
          ]),
          _cache[100] || (_cache[100] = _createElementVNode("div", { class: "protocol-strip" }, [
            _createElementVNode("span", null, "OpenAI"),
            _createElementVNode("span", null, "Anthropic"),
            _createElementVNode("span", null, "Gemini")
          ], -1 /* CACHED */))
        ]),
        _createElementVNode("div", _hoisted_17, [
          (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.metricCards, (card) => {
            return (_openBlock(), _createElementBlock("article", {
              class: "metric",
              key: card.label
            }, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t(card.label)), 1 /* TEXT */),
              _createElementVNode("strong", null, _toDisplayString(card.value), 1 /* TEXT */)
            ]))
          }), 128 /* KEYED_FRAGMENT */))
        ]),
        _createElementVNode("div", _hoisted_18, [
          _createElementVNode("article", _hoisted_19, [
            _createElementVNode("div", _hoisted_20, [
              _createElementVNode("h2", null, _toDisplayString(_ctx.t('nav.providers')), 1 /* TEXT */),
              _createElementVNode("div", _hoisted_21, [
                _createElementVNode("button", {
                  class: "ghost",
                  type: "button",
                  onClick: _cache[5] || (_cache[5] = (...args) => (_ctx.openProviderForm && _ctx.openProviderForm(...args)))
                }, _toDisplayString(_ctx.t('action.addProvider')), 1 /* TEXT */),
                _createElementVNode("button", {
                  class: "ghost",
                  type: "button",
                  onClick: _cache[6] || (_cache[6] = $event => (_ctx.openKeyForm()))
                }, _toDisplayString(_ctx.t('action.addKey')), 1 /* TEXT */)
              ])
            ]),
            _createElementVNode("div", _hoisted_22, [
              (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (p) => {
                return (_openBlock(), _createElementBlock("div", {
                  key: p.id,
                  class: "compact-row"
                }, [
                  _createElementVNode("div", null, [
                    _createElementVNode("strong", null, _toDisplayString(p.name), 1 /* TEXT */),
                    _createElementVNode("small", null, _toDisplayString(_ctx.providerTypeLabel(p.type)) + " · " + _toDisplayString(p.baseUrl), 1 /* TEXT */)
                  ]),
                  _createElementVNode("span", {
                    class: _normalizeClass(['pill', p.enabled ? 'ok' : 'bad'])
                  }, _toDisplayString(p.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */)
                ]))
              }), 128 /* KEYED_FRAGMENT */)),
              (!_ctx.providers.length)
                ? (_openBlock(), _createElementBlock("div", _hoisted_23, [
                    _createElementVNode("strong", null, _toDisplayString(_ctx.t('empty.providers')), 1 /* TEXT */),
                    _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.providersHint')), 1 /* TEXT */)
                  ]))
                : _createCommentVNode("v-if", true)
            ])
          ]),
          _createElementVNode("article", _hoisted_24, [
            _createElementVNode("div", _hoisted_25, [
              _createElementVNode("h2", null, _toDisplayString(_ctx.t('dashboard.recent')), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('dashboard.recentCount', { count: _ctx.recentLogs.length })), 1 /* TEXT */)
            ]),
            _createElementVNode("div", _hoisted_26, [
              (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.recentLogs, (log) => {
                return (_openBlock(), _createElementBlock("div", {
                  key: log.id,
                  class: _normalizeClass(['log-row', 'traffic-row', _ctx.logStatusClass(log.status)])
                }, [
                  _createElementVNode("span", _hoisted_27, _toDisplayString(_ctx.logStatusLabel(log.status)), 1 /* TEXT */),
                  _cache[101] || (_cache[101] = _createElementVNode("span", {
                    class: "traffic-marker",
                    "aria-hidden": "true"
                  }, null, -1 /* CACHED */)),
                  _createElementVNode("div", _hoisted_28, [
                    _createElementVNode("span", _hoisted_29, _toDisplayString(log.status || '-'), 1 /* TEXT */),
                    _createElementVNode("strong", _hoisted_30, _toDisplayString(_ctx.logProtocolLabel(log.inboundProtocol)), 1 /* TEXT */),
                    _createElementVNode("span", _hoisted_31, _toDisplayString(log.latencyMs) + " ms", 1 /* TEXT */),
                    _createElementVNode("time", {
                      class: "traffic-time",
                      datetime: log.createdAt
                    }, _toDisplayString(_ctx.formatShortTime(log.createdAt)), 9 /* TEXT, PROPS */, _hoisted_32)
                  ]),
                  _createElementVNode("div", _hoisted_33, [
                    _createElementVNode("span", {
                      class: _normalizeClass(['traffic-model', { muted: !log.model || String(log.model).toLowerCase() === 'auto' }])
                    }, _toDisplayString(_ctx.logModelLabel(log.model)), 3 /* TEXT, CLASS */),
                    (log.upstreamModel && log.upstreamModel !== log.model)
                      ? (_openBlock(), _createElementBlock("span", _hoisted_34, _toDisplayString(log.upstreamModel), 1 /* TEXT */))
                      : _createCommentVNode("v-if", true),
                    (log.attempts > 1)
                      ? (_openBlock(), _createElementBlock("span", _hoisted_35, _toDisplayString(log.attempts) + " " + _toDisplayString(_ctx.t('logs.attempts')), 1 /* TEXT */))
                      : _createCommentVNode("v-if", true),
                    (log.providerId)
                      ? (_openBlock(), _createElementBlock("span", _hoisted_36, _toDisplayString(log.providerId), 1 /* TEXT */))
                      : _createCommentVNode("v-if", true),
                    (log.errorType)
                      ? (_openBlock(), _createElementBlock("span", _hoisted_37, _toDisplayString(log.errorType), 1 /* TEXT */))
                      : _createCommentVNode("v-if", true)
                  ])
                ], 2 /* CLASS */))
              }), 128 /* KEYED_FRAGMENT */)),
              (!_ctx.recentLogs.length)
                ? (_openBlock(), _createElementBlock("div", _hoisted_38, [
                    _createElementVNode("strong", null, _toDisplayString(_ctx.t('empty.logs')), 1 /* TEXT */),
                    _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.logsHint')), 1 /* TEXT */)
                  ]))
                : _createCommentVNode("v-if", true)
            ])
          ])
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'dashboard']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_39, [
        _createElementVNode("article", _hoisted_40, [
          _createElementVNode("div", _hoisted_41, [
            _createElementVNode("div", _hoisted_42, [
              _createElementVNode("button", {
                class: "ghost",
                type: "button",
                onClick: _cache[7] || (_cache[7] = (...args) => (_ctx.refreshBalances && _ctx.refreshBalances(...args))),
                disabled: _ctx.balanceRefreshing
              }, _toDisplayString(_ctx.balanceRefreshing ? _ctx.t('action.refreshingBalance') : _ctx.t('action.refreshBalance')), 9 /* TEXT, PROPS */, _hoisted_43),
              _createElementVNode("button", {
                type: "button",
                onClick: _cache[8] || (_cache[8] = (...args) => (_ctx.openProviderForm && _ctx.openProviderForm(...args)))
              }, _toDisplayString(_ctx.t('action.addProvider')), 1 /* TEXT */)
            ])
          ]),
          _createElementVNode("div", _hoisted_44, [
            _createElementVNode("div", _hoisted_45, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('provider.summaryProviders')), 1 /* TEXT */),
              _createElementVNode("strong", null, _toDisplayString(_ctx.providers.length), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.summaryProvidersCopy')), 1 /* TEXT */)
            ]),
            _createElementVNode("div", _hoisted_46, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('provider.summaryKeys')), 1 /* TEXT */),
              _createElementVNode("strong", null, _toDisplayString(_ctx.keys.length), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.summaryKeysCopy')), 1 /* TEXT */)
            ]),
            _createElementVNode("div", _hoisted_47, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('provider.summaryEndpoint')), 1 /* TEXT */),
              _createElementVNode("code", null, _toDisplayString(_ctx.dashboard?.service?.proxyUrl || 'http://localhost:18787'), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.summaryEndpointCopy')), 1 /* TEXT */)
            ])
          ]),
          (_ctx.providerFormOpen && !_ctx.providerForm.editingId)
            ? (_openBlock(), _createElementBlock("form", {
                key: 0,
                class: "form-grid top-create-form",
                "data-testid": "provider-form",
                onSubmit: _cache[23] || (_cache[23] = _withModifiers((...args) => (_ctx.saveProvider && _ctx.saveProvider(...args)), ["prevent"]))
              }, [
                (_ctx.formError)
                  ? (_openBlock(), _createElementBlock("div", _hoisted_48, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                  : _createCommentVNode("v-if", true),
                _withDirectives(_createElementVNode("input", {
                  "onUpdate:modelValue": _cache[9] || (_cache[9] = $event => ((_ctx.providerForm.id) = $event)),
                  disabled: !!_ctx.providerForm.editingId,
                  placeholder: _ctx.t('placeholder.providerId')
                }, null, 8 /* PROPS */, _hoisted_49), [
                  [
                    _vModelText,
                    _ctx.providerForm.id,
                    void 0,
                    { trim: true }
                  ]
                ]),
                _withDirectives(_createElementVNode("input", {
                  "onUpdate:modelValue": _cache[10] || (_cache[10] = $event => ((_ctx.providerForm.name) = $event)),
                  placeholder: _ctx.t('placeholder.name')
                }, null, 8 /* PROPS */, _hoisted_50), [
                  [
                    _vModelText,
                    _ctx.providerForm.name,
                    void 0,
                    { trim: true }
                  ]
                ]),
                _withDirectives(_createElementVNode("select", {
                  "onUpdate:modelValue": _cache[11] || (_cache[11] = $event => ((_ctx.providerForm.type) = $event))
                }, [
                  (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerTypes, (type) => {
                    return (_openBlock(), _createElementBlock("option", {
                      key: type.value,
                      value: type.value
                    }, _toDisplayString(type.label), 9 /* TEXT, PROPS */, _hoisted_51))
                  }), 128 /* KEYED_FRAGMENT */))
                ], 512 /* NEED_PATCH */), [
                  [_vModelSelect, _ctx.providerForm.type]
                ]),
                _withDirectives(_createElementVNode("input", {
                  "onUpdate:modelValue": _cache[12] || (_cache[12] = $event => ((_ctx.providerForm.baseUrl) = $event)),
                  placeholder: _ctx.t('placeholder.baseUrl')
                }, null, 8 /* PROPS */, _hoisted_52), [
                  [
                    _vModelText,
                    _ctx.providerForm.baseUrl,
                    void 0,
                    { trim: true }
                  ]
                ]),
                _withDirectives(_createElementVNode("input", {
                  "onUpdate:modelValue": _cache[13] || (_cache[13] = $event => ((_ctx.providerForm.balancePath) = $event)),
                  placeholder: _ctx.t('placeholder.balancePath')
                }, null, 8 /* PROPS */, _hoisted_53), [
                  [
                    _vModelText,
                    _ctx.providerForm.balancePath,
                    void 0,
                    { trim: true }
                  ]
                ]),
                _withDirectives(_createElementVNode("input", {
                  "onUpdate:modelValue": _cache[14] || (_cache[14] = $event => ((_ctx.providerForm.priority) = $event)),
                  type: "number",
                  min: "0",
                  max: "1000",
                  placeholder: _ctx.t('placeholder.priority')
                }, null, 8 /* PROPS */, _hoisted_54), [
                  [
                    _vModelText,
                    _ctx.providerForm.priority,
                    void 0,
                    { number: true }
                  ]
                ]),
                _createElementVNode("div", _hoisted_55, [
                  _withDirectives(_createElementVNode("select", {
                    "onUpdate:modelValue": _cache[15] || (_cache[15] = $event => ((_ctx.providerForm.defaultModel) = $event))
                  }, [
                    _createElementVNode("option", _hoisted_56, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                    (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.providerForm.id || _ctx.providerForm.editingId), (model) => {
                      return (_openBlock(), _createElementBlock("option", {
                        key: model,
                        value: model
                      }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_57))
                    }), 128 /* KEYED_FRAGMENT */))
                  ], 512 /* NEED_PATCH */), [
                    [_vModelSelect, _ctx.providerForm.defaultModel]
                  ]),
                  _createElementVNode("small", null, _toDisplayString(_ctx.t('model.providerHint')), 1 /* TEXT */)
                ]),
                _createElementVNode("div", _hoisted_58, [
                  (!_ctx.providerForm.editingId)
                    ? (_openBlock(), _createElementBlock(_Fragment, { key: 0 }, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.t('provider.firstKeyTitle')), 1 /* TEXT */),
                        _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.firstKeyCopy')), 1 /* TEXT */)
                      ], 64 /* STABLE_FRAGMENT */))
                    : (_openBlock(), _createElementBlock(_Fragment, { key: 1 }, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.t('action.editProvider')), 1 /* TEXT */),
                        _createElementVNode("small", null, _toDisplayString(_ctx.providerForm.baseUrl), 1 /* TEXT */)
                      ], 64 /* STABLE_FRAGMENT */))
                ]),
                (!_ctx.providerForm.editingId)
                  ? _withDirectives((_openBlock(), _createElementBlock("input", {
                      key: 1,
                      "onUpdate:modelValue": _cache[16] || (_cache[16] = $event => ((_ctx.providerForm.firstKeyName) = $event)),
                      placeholder: _ctx.t('placeholder.firstKeyName')
                    }, null, 8 /* PROPS */, _hoisted_59)), [
                      [
                        _vModelText,
                        _ctx.providerForm.firstKeyName,
                        void 0,
                        { trim: true }
                      ]
                    ])
                  : _createCommentVNode("v-if", true),
                (!_ctx.providerForm.editingId)
                  ? _withDirectives((_openBlock(), _createElementBlock("input", {
                      key: 2,
                      "onUpdate:modelValue": _cache[17] || (_cache[17] = $event => ((_ctx.providerForm.firstKeySecret) = $event)),
                      type: "password",
                      placeholder: _ctx.t('placeholder.firstKeySecret'),
                      autocomplete: "off"
                    }, null, 8 /* PROPS */, _hoisted_60)), [
                      [
                        _vModelText,
                        _ctx.providerForm.firstKeySecret,
                        void 0,
                        { trim: true }
                      ]
                    ])
                  : _createCommentVNode("v-if", true),
                (!_ctx.providerForm.editingId)
                  ? _withDirectives((_openBlock(), _createElementBlock("input", {
                      key: 3,
                      "onUpdate:modelValue": _cache[18] || (_cache[18] = $event => ((_ctx.providerForm.firstKeyPriority) = $event)),
                      type: "number",
                      min: "0",
                      max: "1000",
                      placeholder: _ctx.t('placeholder.keyPriority')
                    }, null, 8 /* PROPS */, _hoisted_61)), [
                      [
                        _vModelText,
                        _ctx.providerForm.firstKeyPriority,
                        void 0,
                        { number: true }
                      ]
                    ])
                  : _createCommentVNode("v-if", true),
                _createElementVNode("details", _hoisted_62, [
                  _createElementVNode("summary", null, _toDisplayString(_ctx.t('provider.modelMapTitle')), 1 /* TEXT */),
                  _createElementVNode("p", null, _toDisplayString(_ctx.t('provider.modelMapCopy')), 1 /* TEXT */),
                  _withDirectives(_createElementVNode("textarea", {
                    "onUpdate:modelValue": _cache[19] || (_cache[19] = $event => ((_ctx.providerForm.modelMap) = $event)),
                    spellcheck: "false",
                    placeholder: _ctx.t('placeholder.modelMap')
                  }, null, 8 /* PROPS */, _hoisted_63), [
                    [_vModelText, _ctx.providerForm.modelMap]
                  ]),
                  _createElementVNode("div", _hoisted_64, [
                    _createElementVNode("button", {
                      class: "ghost",
                      type: "button",
                      onClick: _cache[20] || (_cache[20] = $event => (_ctx.providerForm.modelMap = '{}'))
                    }, _toDisplayString(_ctx.t('action.emptyMap')), 1 /* TEXT */),
                    _createElementVNode("button", {
                      class: "ghost",
                      type: "button",
                      onClick: _cache[21] || (_cache[21] = $event => (_ctx.providerForm.modelMap = _ctx.modelMapExample))
                    }, _toDisplayString(_ctx.t('action.exampleMap')), 1 /* TEXT */)
                  ])
                ]),
                _createElementVNode("div", _hoisted_65, [
                  _createElementVNode("button", {
                    class: "ghost",
                    type: "button",
                    "data-testid": "cancel-provider",
                    onClick: _cache[22] || (_cache[22] = (...args) => (_ctx.cancelProviderForm && _ctx.cancelProviderForm(...args)))
                  }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                  _createElementVNode("button", _hoisted_66, _toDisplayString(_ctx.t(_ctx.providerForm.editingId ? 'action.updateProvider' : 'action.saveProvider')), 1 /* TEXT */)
                ])
              ], 32 /* NEED_HYDRATION */))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_67, [
            (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (p) => {
              return (_openBlock(), _createElementBlock("article", {
                key: p.id,
                class: "provider-card"
              }, [
                _createElementVNode("div", _hoisted_68, [
                  _createElementVNode("div", null, [
                    _createElementVNode("div", _hoisted_69, [
                      _createElementVNode("code", null, _toDisplayString(p.id), 1 /* TEXT */),
                      _createElementVNode("span", {
                        class: _normalizeClass(['pill', p.enabled ? 'ok' : 'bad'])
                      }, _toDisplayString(p.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */),
                      _createElementVNode("span", _hoisted_70, _toDisplayString(_ctx.providerTypeLabel(p.type)), 1 /* TEXT */)
                    ]),
                    _createElementVNode("h3", null, _toDisplayString(p.name), 1 /* TEXT */),
                    _createElementVNode("small", _hoisted_71, _toDisplayString(p.baseUrl), 1 /* TEXT */)
                  ]),
                  _createElementVNode("div", _hoisted_72, [
                    _createElementVNode("button", {
                      type: "button",
                      onClick: $event => (_ctx.openKeyForm(p))
                    }, _toDisplayString(_ctx.t('action.addKey')), 9 /* TEXT, PROPS */, _hoisted_73),
                    _createElementVNode("button", {
                      class: "ghost",
                      type: "button",
                      onClick: $event => (_ctx.editProvider(p))
                    }, _toDisplayString(_ctx.t('action.editProvider')), 9 /* TEXT, PROPS */, _hoisted_74),
                    _createElementVNode("button", {
                      class: "ghost",
                      type: "button",
                      onClick: $event => (_ctx.toggleProvider(p))
                    }, _toDisplayString(p.enabled ? _ctx.t('action.disable') : _ctx.t('action.enable')), 9 /* TEXT, PROPS */, _hoisted_75),
                    _createElementVNode("button", {
                      class: "danger",
                      type: "button",
                      onClick: $event => (_ctx.deleteProvider(p))
                    }, _toDisplayString(_ctx.t('action.delete')), 9 /* TEXT, PROPS */, _hoisted_76)
                  ])
                ]),
                (_ctx.providerFormOpen && _ctx.providerForm.editingId === p.id)
                  ? (_openBlock(), _createElementBlock("form", {
                      key: 0,
                      class: "form-grid inline-edit-form provider-inline-form",
                      "data-testid": "provider-form",
                      onSubmit: _cache[35] || (_cache[35] = _withModifiers((...args) => (_ctx.saveProvider && _ctx.saveProvider(...args)), ["prevent"]))
                    }, [
                      _createElementVNode("div", _hoisted_77, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.t('action.editProvider')), 1 /* TEXT */),
                        _createElementVNode("small", null, _toDisplayString(p.name) + " · " + _toDisplayString(p.baseUrl), 1 /* TEXT */)
                      ]),
                      (_ctx.formError)
                        ? (_openBlock(), _createElementBlock("div", _hoisted_78, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                        : _createCommentVNode("v-if", true),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[24] || (_cache[24] = $event => ((_ctx.providerForm.id) = $event)),
                        disabled: "",
                        placeholder: _ctx.t('placeholder.providerId')
                      }, null, 8 /* PROPS */, _hoisted_79), [
                        [
                          _vModelText,
                          _ctx.providerForm.id,
                          void 0,
                          { trim: true }
                        ]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[25] || (_cache[25] = $event => ((_ctx.providerForm.name) = $event)),
                        placeholder: _ctx.t('placeholder.name')
                      }, null, 8 /* PROPS */, _hoisted_80), [
                        [
                          _vModelText,
                          _ctx.providerForm.name,
                          void 0,
                          { trim: true }
                        ]
                      ]),
                      _withDirectives(_createElementVNode("select", {
                        "onUpdate:modelValue": _cache[26] || (_cache[26] = $event => ((_ctx.providerForm.type) = $event))
                      }, [
                        (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerTypes, (type) => {
                          return (_openBlock(), _createElementBlock("option", {
                            key: type.value,
                            value: type.value
                          }, _toDisplayString(type.label), 9 /* TEXT, PROPS */, _hoisted_81))
                        }), 128 /* KEYED_FRAGMENT */))
                      ], 512 /* NEED_PATCH */), [
                        [_vModelSelect, _ctx.providerForm.type]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[27] || (_cache[27] = $event => ((_ctx.providerForm.baseUrl) = $event)),
                        placeholder: _ctx.t('placeholder.baseUrl')
                      }, null, 8 /* PROPS */, _hoisted_82), [
                        [
                          _vModelText,
                          _ctx.providerForm.baseUrl,
                          void 0,
                          { trim: true }
                        ]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[28] || (_cache[28] = $event => ((_ctx.providerForm.balancePath) = $event)),
                        placeholder: _ctx.t('placeholder.balancePath')
                      }, null, 8 /* PROPS */, _hoisted_83), [
                        [
                          _vModelText,
                          _ctx.providerForm.balancePath,
                          void 0,
                          { trim: true }
                        ]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[29] || (_cache[29] = $event => ((_ctx.providerForm.priority) = $event)),
                        type: "number",
                        min: "0",
                        max: "1000",
                        placeholder: _ctx.t('placeholder.priority')
                      }, null, 8 /* PROPS */, _hoisted_84), [
                        [
                          _vModelText,
                          _ctx.providerForm.priority,
                          void 0,
                          { number: true }
                        ]
                      ]),
                      _createElementVNode("div", _hoisted_85, [
                        _withDirectives(_createElementVNode("select", {
                          "onUpdate:modelValue": _cache[30] || (_cache[30] = $event => ((_ctx.providerForm.defaultModel) = $event))
                        }, [
                          _createElementVNode("option", _hoisted_86, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                          (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.providerForm.id || _ctx.providerForm.editingId), (model) => {
                            return (_openBlock(), _createElementBlock("option", {
                              key: model,
                              value: model
                            }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_87))
                          }), 128 /* KEYED_FRAGMENT */))
                        ], 512 /* NEED_PATCH */), [
                          [_vModelSelect, _ctx.providerForm.defaultModel]
                        ]),
                        _createElementVNode("small", null, _toDisplayString(_ctx.t('model.providerHint')), 1 /* TEXT */)
                      ]),
                      _createElementVNode("details", _hoisted_88, [
                        _createElementVNode("summary", null, _toDisplayString(_ctx.t('provider.modelMapTitle')), 1 /* TEXT */),
                        _createElementVNode("p", null, _toDisplayString(_ctx.t('provider.modelMapCopy')), 1 /* TEXT */),
                        _withDirectives(_createElementVNode("textarea", {
                          "onUpdate:modelValue": _cache[31] || (_cache[31] = $event => ((_ctx.providerForm.modelMap) = $event)),
                          spellcheck: "false",
                          placeholder: _ctx.t('placeholder.modelMap')
                        }, null, 8 /* PROPS */, _hoisted_89), [
                          [_vModelText, _ctx.providerForm.modelMap]
                        ]),
                        _createElementVNode("div", _hoisted_90, [
                          _createElementVNode("button", {
                            class: "ghost",
                            type: "button",
                            onClick: _cache[32] || (_cache[32] = $event => (_ctx.providerForm.modelMap = '{}'))
                          }, _toDisplayString(_ctx.t('action.emptyMap')), 1 /* TEXT */),
                          _createElementVNode("button", {
                            class: "ghost",
                            type: "button",
                            onClick: _cache[33] || (_cache[33] = $event => (_ctx.providerForm.modelMap = _ctx.modelMapExample))
                          }, _toDisplayString(_ctx.t('action.exampleMap')), 1 /* TEXT */)
                        ])
                      ]),
                      _createElementVNode("div", _hoisted_91, [
                        _createElementVNode("button", {
                          class: "ghost",
                          type: "button",
                          "data-testid": "cancel-provider",
                          onClick: _cache[34] || (_cache[34] = (...args) => (_ctx.cancelProviderForm && _ctx.cancelProviderForm(...args)))
                        }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                        _createElementVNode("button", _hoisted_92, _toDisplayString(_ctx.t('action.updateProvider')), 1 /* TEXT */)
                      ])
                    ], 32 /* NEED_HYDRATION */))
                  : _createCommentVNode("v-if", true),
                _createElementVNode("div", _hoisted_93, [
                  _createElementVNode("div", null, [
                    _createElementVNode("span", null, _toDisplayString(_ctx.t('table.priority')), 1 /* TEXT */),
                    _createElementVNode("strong", null, _toDisplayString(p.priority), 1 /* TEXT */)
                  ]),
                  _createElementVNode("div", null, [
                    _createElementVNode("span", null, _toDisplayString(_ctx.t('table.keys')), 1 /* TEXT */),
                    _createElementVNode("strong", null, _toDisplayString(_ctx.providerKeys(p.id).length), 1 /* TEXT */)
                  ]),
                  _createElementVNode("div", null, [
                    _createElementVNode("span", null, _toDisplayString(_ctx.t('model.currentDefault')), 1 /* TEXT */),
                    _createElementVNode("strong", null, _toDisplayString(_ctx.modelMapDefault(p.modelMap) || _ctx.t('model.useRequestDefault')), 1 /* TEXT */)
                  ]),
                  _createElementVNode("div", null, [
                    _createElementVNode("span", null, _toDisplayString(_ctx.t('table.balance')), 1 /* TEXT */),
                    _createElementVNode("strong", {
                      class: _normalizeClass(_ctx.providerBalanceClass(p.id))
                    }, _toDisplayString(_ctx.providerBalanceText(p.id)), 3 /* TEXT, CLASS */)
                  ]),
                  _createElementVNode("div", null, [
                    _createElementVNode("span", null, _toDisplayString(_ctx.t('provider.apiEndpoint')), 1 /* TEXT */),
                    _createElementVNode("code", null, _toDisplayString(_ctx.providerApiEndpoint(p)), 1 /* TEXT */)
                  ])
                ]),
                _createElementVNode("div", _hoisted_94, [
                  _createElementVNode("div", _hoisted_95, [
                    _createElementVNode("div", null, [
                      _createElementVNode("div", _hoisted_96, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.t('model.inventory')), 1 /* TEXT */),
                        _createElementVNode("span", {
                          class: _normalizeClass(['pill', _ctx.discoveryStatusClass(p.id)])
                        }, _toDisplayString(_ctx.discoveryStatusText(p.id)), 3 /* TEXT, CLASS */),
                        _createElementVNode("span", _hoisted_97, _toDisplayString(_ctx.providerDiscoveredModels(p.id).length), 1 /* TEXT */)
                      ]),
                      (_ctx.discoveryForProvider(p.id).lastSuccessAt)
                        ? (_openBlock(), _createElementBlock("small", _hoisted_98, _toDisplayString(_ctx.t('model.lastDiscovered')) + " " + _toDisplayString(_ctx.formatGatewayTime(_ctx.discoveryForProvider(p.id).lastSuccessAt)), 1 /* TEXT */))
                        : (_openBlock(), _createElementBlock("small", _hoisted_99, _toDisplayString(_ctx.t('model.notDiscovered')), 1 /* TEXT */))
                    ]),
                    _createElementVNode("button", {
                      class: "ghost",
                      type: "button",
                      disabled: _ctx.refreshingProviderId === p.id || !_ctx.providerKeys(p.id).length,
                      onClick: $event => (_ctx.refreshProviderModels(p))
                    }, _toDisplayString(_ctx.refreshingProviderId === p.id ? _ctx.t('action.discoveringModels') : _ctx.t('action.refreshModels')), 9 /* TEXT, PROPS */, _hoisted_100)
                  ]),
                  (_ctx.providerModelPreview(p.id).length)
                    ? (_openBlock(), _createElementBlock("div", _hoisted_101, [
                        (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelPreview(p.id), (model) => {
                          return (_openBlock(), _createElementBlock("code", {
                            key: model,
                            class: "model-chip"
                          }, _toDisplayString(model), 1 /* TEXT */))
                        }), 128 /* KEYED_FRAGMENT */)),
                        (_ctx.providerDiscoveredModels(p.id).length > _ctx.providerModelPreview(p.id).length)
                          ? (_openBlock(), _createElementBlock("span", _hoisted_102, "+" + _toDisplayString(_ctx.providerDiscoveredModels(p.id).length - _ctx.providerModelPreview(p.id).length), 1 /* TEXT */))
                          : _createCommentVNode("v-if", true)
                      ]))
                    : (_ctx.discoveryForProvider(p.id).lastError)
                      ? (_openBlock(), _createElementBlock("small", _hoisted_103, _toDisplayString(_ctx.discoveryForProvider(p.id).lastError), 1 /* TEXT */))
                      : (_openBlock(), _createElementBlock("small", _hoisted_104, _toDisplayString(_ctx.t('model.inventoryEmpty')), 1 /* TEXT */))
                ]),
                (_ctx.keyFormOpen && !_ctx.keyForm.editingId && _ctx.keyForm.providerId === p.id)
                  ? (_openBlock(), _createElementBlock("form", {
                      key: 1,
                      class: "form-grid inline-edit-form key-inline-form",
                      "data-testid": "key-form",
                      onSubmit: _cache[43] || (_cache[43] = _withModifiers((...args) => (_ctx.saveKey && _ctx.saveKey(...args)), ["prevent"]))
                    }, [
                      _createElementVNode("div", _hoisted_105, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.t('action.addKey')), 1 /* TEXT */),
                        _createElementVNode("small", null, _toDisplayString(p.name) + " · " + _toDisplayString(p.baseUrl), 1 /* TEXT */)
                      ]),
                      (_ctx.formError)
                        ? (_openBlock(), _createElementBlock("div", _hoisted_106, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                        : _createCommentVNode("v-if", true),
                      _withDirectives(_createElementVNode("select", {
                        "onUpdate:modelValue": _cache[36] || (_cache[36] = $event => ((_ctx.keyForm.providerId) = $event)),
                        disabled: ""
                      }, [
                        (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (provider) => {
                          return (_openBlock(), _createElementBlock("option", {
                            key: provider.id,
                            value: provider.id
                          }, _toDisplayString(provider.name) + " (" + _toDisplayString(provider.id) + ")", 9 /* TEXT, PROPS */, _hoisted_107))
                        }), 128 /* KEYED_FRAGMENT */))
                      ], 512 /* NEED_PATCH */), [
                        [_vModelSelect, _ctx.keyForm.providerId]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[37] || (_cache[37] = $event => ((_ctx.keyForm.name) = $event)),
                        placeholder: _ctx.t('placeholder.keyDisplayName')
                      }, null, 8 /* PROPS */, _hoisted_108), [
                        [
                          _vModelText,
                          _ctx.keyForm.name,
                          void 0,
                          { trim: true }
                        ]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[38] || (_cache[38] = $event => ((_ctx.keyForm.secret) = $event)),
                        type: "password",
                        placeholder: _ctx.t('placeholder.secret'),
                        autocomplete: "off"
                      }, null, 8 /* PROPS */, _hoisted_109), [
                        [
                          _vModelText,
                          _ctx.keyForm.secret,
                          void 0,
                          { trim: true }
                        ]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[39] || (_cache[39] = $event => ((_ctx.keyForm.priority) = $event)),
                        type: "number",
                        min: "0",
                        max: "1000",
                        placeholder: _ctx.t('placeholder.priority')
                      }, null, 8 /* PROPS */, _hoisted_110), [
                        [
                          _vModelText,
                          _ctx.keyForm.priority,
                          void 0,
                          { number: true }
                        ]
                      ]),
                      _createElementVNode("div", _hoisted_111, [
                        _withDirectives(_createElementVNode("select", {
                          "onUpdate:modelValue": _cache[40] || (_cache[40] = $event => ((_ctx.keyForm.defaultModel) = $event))
                        }, [
                          _createElementVNode("option", _hoisted_112, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                          (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.keyForm.providerId), (model) => {
                            return (_openBlock(), _createElementBlock("option", {
                              key: model,
                              value: model
                            }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_113))
                          }), 128 /* KEYED_FRAGMENT */))
                        ], 512 /* NEED_PATCH */), [
                          [_vModelSelect, _ctx.keyForm.defaultModel]
                        ]),
                        _createElementVNode("small", null, _toDisplayString(_ctx.t('model.keyHint')), 1 /* TEXT */)
                      ]),
                      _createElementVNode("label", _hoisted_114, [
                        _withDirectives(_createElementVNode("input", {
                          "onUpdate:modelValue": _cache[41] || (_cache[41] = $event => ((_ctx.keyForm.enabled) = $event)),
                          type: "checkbox"
                        }, null, 512 /* NEED_PATCH */), [
                          [_vModelCheckbox, _ctx.keyForm.enabled]
                        ]),
                        _createTextVNode(" " + _toDisplayString(_ctx.t('status.enabled')), 1 /* TEXT */)
                      ]),
                      _createElementVNode("div", _hoisted_115, [
                        _createElementVNode("button", {
                          class: "ghost",
                          type: "button",
                          "data-testid": "cancel-key",
                          onClick: _cache[42] || (_cache[42] = (...args) => (_ctx.cancelKeyForm && _ctx.cancelKeyForm(...args)))
                        }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                        _createElementVNode("button", _hoisted_116, _toDisplayString(_ctx.t('action.saveKey')), 1 /* TEXT */)
                      ])
                    ], 32 /* NEED_HYDRATION */))
                  : _createCommentVNode("v-if", true),
                (_ctx.providerKeys(p.id).length)
                  ? (_openBlock(), _createElementBlock("div", _hoisted_117, [
                      (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerKeys(p.id), (k) => {
                        return (_openBlock(), _createElementBlock("div", {
                          key: k.id,
                          class: "key-card"
                        }, [
                          _createElementVNode("div", _hoisted_118, [
                            _createElementVNode("div", null, [
                              _createElementVNode("div", _hoisted_119, [
                                _createElementVNode("strong", _hoisted_120, _toDisplayString(k.name || k.id), 1 /* TEXT */),
                                _createElementVNode("span", _hoisted_121, _toDisplayString(_ctx.t('table.priority')) + " " + _toDisplayString(k.priority), 1 /* TEXT */),
                                (k.manualPreferred)
                                  ? (_openBlock(), _createElementBlock("span", _hoisted_122, _toDisplayString(_ctx.t('status.preferred')), 1 /* TEXT */))
                                  : _createCommentVNode("v-if", true),
                                _createElementVNode("span", {
                                  class: _normalizeClass(['pill', k.enabled ? 'ok' : 'bad'])
                                }, _toDisplayString(k.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */)
                              ]),
                              _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.keyHealth', { ok: k.successCount || 0, fail: k.failureCount || 0 })), 1 /* TEXT */)
                            ]),
                            _createElementVNode("code", _hoisted_123, _toDisplayString(k.keyHint), 1 /* TEXT */)
                          ]),
                          (_ctx.keyFormOpen && _ctx.keyForm.editingId === k.id)
                            ? (_openBlock(), _createElementBlock("form", {
                                key: 0,
                                class: "form-grid inline-edit-form key-inline-form",
                                "data-testid": "key-form",
                                onSubmit: _cache[51] || (_cache[51] = _withModifiers((...args) => (_ctx.saveKey && _ctx.saveKey(...args)), ["prevent"]))
                              }, [
                                _createElementVNode("div", _hoisted_124, [
                                  _createElementVNode("strong", null, _toDisplayString(_ctx.t('action.editKey')), 1 /* TEXT */),
                                  _createElementVNode("small", {
                                    class: "key-edit-context",
                                    title: k.id
                                  }, _toDisplayString(p.name) + " · " + _toDisplayString(_ctx.t('table.internalId')) + ": " + _toDisplayString(k.id), 9 /* TEXT, PROPS */, _hoisted_125)
                                ]),
                                (_ctx.formError)
                                  ? (_openBlock(), _createElementBlock("div", _hoisted_126, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                                  : _createCommentVNode("v-if", true),
                                _withDirectives(_createElementVNode("select", {
                                  "onUpdate:modelValue": _cache[44] || (_cache[44] = $event => ((_ctx.keyForm.providerId) = $event)),
                                  disabled: ""
                                }, [
                                  (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (provider) => {
                                    return (_openBlock(), _createElementBlock("option", {
                                      key: provider.id,
                                      value: provider.id
                                    }, _toDisplayString(provider.name) + " (" + _toDisplayString(provider.id) + ")", 9 /* TEXT, PROPS */, _hoisted_127))
                                  }), 128 /* KEYED_FRAGMENT */))
                                ], 512 /* NEED_PATCH */), [
                                  [_vModelSelect, _ctx.keyForm.providerId]
                                ]),
                                _withDirectives(_createElementVNode("input", {
                                  "onUpdate:modelValue": _cache[45] || (_cache[45] = $event => ((_ctx.keyForm.name) = $event)),
                                  placeholder: _ctx.t('placeholder.keyDisplayName')
                                }, null, 8 /* PROPS */, _hoisted_128), [
                                  [
                                    _vModelText,
                                    _ctx.keyForm.name,
                                    void 0,
                                    { trim: true }
                                  ]
                                ]),
                                _withDirectives(_createElementVNode("input", {
                                  "onUpdate:modelValue": _cache[46] || (_cache[46] = $event => ((_ctx.keyForm.secret) = $event)),
                                  type: "password",
                                  placeholder: _ctx.t('placeholder.secretKeep'),
                                  autocomplete: "off"
                                }, null, 8 /* PROPS */, _hoisted_129), [
                                  [
                                    _vModelText,
                                    _ctx.keyForm.secret,
                                    void 0,
                                    { trim: true }
                                  ]
                                ]),
                                _withDirectives(_createElementVNode("input", {
                                  "onUpdate:modelValue": _cache[47] || (_cache[47] = $event => ((_ctx.keyForm.priority) = $event)),
                                  type: "number",
                                  min: "0",
                                  max: "1000",
                                  placeholder: _ctx.t('placeholder.priority')
                                }, null, 8 /* PROPS */, _hoisted_130), [
                                  [
                                    _vModelText,
                                    _ctx.keyForm.priority,
                                    void 0,
                                    { number: true }
                                  ]
                                ]),
                                _createElementVNode("div", _hoisted_131, [
                                  _withDirectives(_createElementVNode("select", {
                                    "onUpdate:modelValue": _cache[48] || (_cache[48] = $event => ((_ctx.keyForm.defaultModel) = $event))
                                  }, [
                                    _createElementVNode("option", _hoisted_132, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                                    (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.keyForm.providerId), (model) => {
                                      return (_openBlock(), _createElementBlock("option", {
                                        key: model,
                                        value: model
                                      }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_133))
                                    }), 128 /* KEYED_FRAGMENT */))
                                  ], 512 /* NEED_PATCH */), [
                                    [_vModelSelect, _ctx.keyForm.defaultModel]
                                  ]),
                                  _createElementVNode("small", null, _toDisplayString(_ctx.t('model.keyHint')), 1 /* TEXT */)
                                ]),
                                _createElementVNode("label", _hoisted_134, [
                                  _withDirectives(_createElementVNode("input", {
                                    "onUpdate:modelValue": _cache[49] || (_cache[49] = $event => ((_ctx.keyForm.enabled) = $event)),
                                    type: "checkbox"
                                  }, null, 512 /* NEED_PATCH */), [
                                    [_vModelCheckbox, _ctx.keyForm.enabled]
                                  ]),
                                  _createTextVNode(" " + _toDisplayString(_ctx.t('status.enabled')), 1 /* TEXT */)
                                ]),
                                _createElementVNode("div", _hoisted_135, [
                                  _createElementVNode("button", {
                                    class: "ghost",
                                    type: "button",
                                    "data-testid": "cancel-key",
                                    onClick: _cache[50] || (_cache[50] = (...args) => (_ctx.cancelKeyForm && _ctx.cancelKeyForm(...args)))
                                  }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                                  _createElementVNode("button", _hoisted_136, _toDisplayString(_ctx.t('action.updateKey')), 1 /* TEXT */)
                                ])
                              ], 32 /* NEED_HYDRATION */))
                            : _createCommentVNode("v-if", true),
                          _createElementVNode("div", _hoisted_137, [
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.testKey(k)),
                              disabled: _ctx.testingKeyId === k.id
                            }, _toDisplayString(_ctx.testingKeyId === k.id ? _ctx.t('action.testing') : _ctx.t('action.test')), 9 /* TEXT, PROPS */, _hoisted_138),
                            (_ctx.keyTestResults[k.id])
                              ? (_openBlock(), _createElementBlock("span", {
                                  key: 0,
                                  class: _normalizeClass(['pill', _ctx.testStatusClass(_ctx.keyTestResults[k.id])])
                                }, _toDisplayString(_ctx.testConnectionText(_ctx.keyTestResults[k.id])), 3 /* TEXT, CLASS */))
                              : _createCommentVNode("v-if", true),
                            (_ctx.keyTestResults[k.id])
                              ? (_openBlock(), _createElementBlock("div", {
                                  key: 1,
                                  class: _normalizeClass(['test-metrics', _ctx.testStatusClass(_ctx.keyTestResults[k.id])])
                                }, [
                                  _createElementVNode("span", null, _toDisplayString(_ctx.t('test.http')) + " " + _toDisplayString(_ctx.keyTestResults[k.id].statusCode || '-'), 1 /* TEXT */),
                                  _createElementVNode("span", null, _toDisplayString(_ctx.t('test.latency')) + " " + _toDisplayString(_ctx.keyTestResults[k.id].latencyMs || 0) + "ms", 1 /* TEXT */),
                                  _createElementVNode("span", null, _toDisplayString(_ctx.t('test.models')) + " " + _toDisplayString(_ctx.keyTestResults[k.id].modelCount ?? '-'), 1 /* TEXT */),
                                  _createElementVNode("span", null, _toDisplayString(_ctx.t('test.tokens')) + " " + _toDisplayString(_ctx.tokenUsageText(_ctx.keyTestResults[k.id])), 1 /* TEXT */)
                                ], 2 /* CLASS */))
                              : _createCommentVNode("v-if", true),
                            (_ctx.testErrorText(_ctx.keyTestResults[k.id]))
                              ? (_openBlock(), _createElementBlock("small", _hoisted_139, _toDisplayString(_ctx.testErrorText(_ctx.keyTestResults[k.id])), 1 /* TEXT */))
                              : _createCommentVNode("v-if", true)
                          ]),
                          _createElementVNode("div", _hoisted_140, [
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.editKey(k))
                            }, _toDisplayString(_ctx.t('action.editKey')), 9 /* TEXT, PROPS */, _hoisted_141),
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.preferKey(k))
                            }, _toDisplayString(_ctx.t('action.prefer')), 9 /* TEXT, PROPS */, _hoisted_142),
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.resetKey(k))
                            }, _toDisplayString(_ctx.t('action.reset')), 9 /* TEXT, PROPS */, _hoisted_143),
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.toggleKey(k))
                            }, _toDisplayString(k.enabled ? _ctx.t('action.disable') : _ctx.t('action.enable')), 9 /* TEXT, PROPS */, _hoisted_144),
                            _createElementVNode("button", {
                              class: "danger",
                              type: "button",
                              onClick: $event => (_ctx.deleteKey(k))
                            }, _toDisplayString(_ctx.t('action.delete')), 9 /* TEXT, PROPS */, _hoisted_145)
                          ])
                        ]))
                      }), 128 /* KEYED_FRAGMENT */))
                    ]))
                  : (_openBlock(), _createElementBlock("div", _hoisted_146, [
                      _createElementVNode("strong", null, _toDisplayString(_ctx.t('empty.keys')), 1 /* TEXT */),
                      _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.keysHint')), 1 /* TEXT */),
                      _createElementVNode("button", {
                        type: "button",
                        onClick: $event => (_ctx.openKeyForm(p))
                      }, _toDisplayString(_ctx.t('action.addKey')), 9 /* TEXT, PROPS */, _hoisted_147)
                    ]))
              ]))
            }), 128 /* KEYED_FRAGMENT */)),
            (!_ctx.providers.length)
              ? (_openBlock(), _createElementBlock("div", _hoisted_148, [
                  _createElementVNode("strong", null, _toDisplayString(_ctx.t('empty.providers')), 1 /* TEXT */),
                  _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.providersHint')), 1 /* TEXT */),
                  _createElementVNode("button", {
                    type: "button",
                    onClick: _cache[52] || (_cache[52] = (...args) => (_ctx.openProviderForm && _ctx.openProviderForm(...args)))
                  }, _toDisplayString(_ctx.t('action.addProvider')), 1 /* TEXT */)
                ]))
              : _createCommentVNode("v-if", true)
          ]),
          _createElementVNode("div", _hoisted_149, [
            _createElementVNode("div", null, [
              _createElementVNode("h2", null, _toDisplayString(_ctx.t('provider.balanceTitle')), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.balancesHint')), 1 /* TEXT */)
            ]),
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              onClick: _cache[53] || (_cache[53] = (...args) => (_ctx.refreshBalances && _ctx.refreshBalances(...args))),
              disabled: _ctx.balanceRefreshing
            }, _toDisplayString(_ctx.balanceRefreshing ? _ctx.t('action.refreshingBalance') : _ctx.t('action.refreshBalance')), 9 /* TEXT, PROPS */, _hoisted_150)
          ]),
          (_ctx.balanceResults.length)
            ? (_openBlock(), _createElementBlock("div", _hoisted_151, [
                (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.balanceResults, (result) => {
                  return (_openBlock(), _createElementBlock("div", {
                    key: result.providerId + ':' + result.keyId,
                    class: "compact-row"
                  }, [
                    _createElementVNode("div", _hoisted_152, [
                      _createElementVNode("strong", null, _toDisplayString(_ctx.providerDisplayName(result.providerId)) + " · " + _toDisplayString(_ctx.keyDisplayName(result.keyId)), 1 /* TEXT */),
                      (_ctx.hasDistinctProviderName(result.providerId) || _ctx.hasDistinctKeyName(result.keyId))
                        ? (_openBlock(), _createElementBlock("small", {
                            key: 0,
                            class: "entity-id",
                            title: `${result.providerId} · ${result.keyId || '-'}`
                          }, _toDisplayString(result.providerId) + " · " + _toDisplayString(result.keyId || '-'), 9 /* TEXT, PROPS */, _hoisted_153))
                        : _createCommentVNode("v-if", true),
                      _createElementVNode("small", null, _toDisplayString(result.error || result.status), 1 /* TEXT */)
                    ]),
                    _createElementVNode("span", {
                      class: _normalizeClass(['pill', _ctx.resultStatusClass(result.status)])
                    }, _toDisplayString(result.status), 3 /* TEXT, CLASS */)
                  ]))
                }), 128 /* KEYED_FRAGMENT */))
              ]))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_154, [
            _createElementVNode("table", null, [
              _createElementVNode("thead", null, [
                _createElementVNode("tr", null, [
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.provider')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.key')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.balance')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.status')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.error')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.updated')), 1 /* TEXT */)
                ])
              ]),
              _createElementVNode("tbody", null, [
                (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.balances, (b) => {
                  return (_openBlock(), _createElementBlock("tr", {
                    key: b.providerId + ':' + b.keyId
                  }, [
                    _createElementVNode("td", null, [
                      _createElementVNode("div", _hoisted_155, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.providerDisplayName(b.providerId)), 1 /* TEXT */),
                        (_ctx.hasDistinctProviderName(b.providerId))
                          ? (_openBlock(), _createElementBlock("code", {
                              key: 0,
                              class: "entity-id",
                              title: b.providerId
                            }, _toDisplayString(b.providerId), 9 /* TEXT, PROPS */, _hoisted_156))
                          : _createCommentVNode("v-if", true)
                      ])
                    ]),
                    _createElementVNode("td", null, [
                      _createElementVNode("div", _hoisted_157, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.keyDisplayName(b.keyId)), 1 /* TEXT */),
                        (_ctx.hasDistinctKeyName(b.keyId))
                          ? (_openBlock(), _createElementBlock("code", {
                              key: 0,
                              class: "entity-id",
                              title: b.keyId
                            }, _toDisplayString(b.keyId), 9 /* TEXT, PROPS */, _hoisted_158))
                          : _createCommentVNode("v-if", true)
                      ])
                    ]),
                    _createElementVNode("td", {
                      class: _normalizeClass(_ctx.balanceValueClass(b))
                    }, _toDisplayString(_ctx.displayBalanceValue(b)), 3 /* TEXT, CLASS */),
                    _createElementVNode("td", null, [
                      _createElementVNode("span", {
                        class: _normalizeClass(['pill', _ctx.balanceStatusClass(b)])
                      }, _toDisplayString(_ctx.balanceStatusText(b.status)), 3 /* TEXT, CLASS */)
                    ]),
                    _createElementVNode("td", null, _toDisplayString(b.error || '-'), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(_ctx.formatGatewayTime(b.refreshedAt || b.updatedAt)), 1 /* TEXT */)
                  ]))
                }), 128 /* KEYED_FRAGMENT */)),
                (!_ctx.balances.length)
                  ? (_openBlock(), _createElementBlock("tr", _hoisted_159, [
                      _createElementVNode("td", _hoisted_160, [
                        _createElementVNode("div", _hoisted_161, [
                          _createElementVNode("strong", null, _toDisplayString(_ctx.t('empty.balances')), 1 /* TEXT */),
                          _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.balancesHint')), 1 /* TEXT */)
                        ])
                      ])
                    ]))
                  : _createCommentVNode("v-if", true)
              ])
            ])
          ])
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'providers']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_162, [
        _createElementVNode("article", _hoisted_163, [
          _createElementVNode("div", _hoisted_164, [
            _createElementVNode("div", null, [
              _createElementVNode("h2", null, _toDisplayString(_ctx.t('modelRouting.title')), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('modelRouting.copy')), 1 /* TEXT */)
            ]),
            _createElementVNode("button", {
              type: "button",
              onClick: _cache[54] || (_cache[54] = $event => (_ctx.openModelRouteForm()))
            }, _toDisplayString(_ctx.t('action.addModelRoute')), 1 /* TEXT */)
          ]),
          (_ctx.modelRouteFormOpen)
            ? (_openBlock(), _createElementBlock("form", {
                key: 0,
                class: "model-route-editor",
                "data-testid": "model-route-form",
                onSubmit: _cache[60] || (_cache[60] = _withModifiers((...args) => (_ctx.saveModelRoute && _ctx.saveModelRoute(...args)), ["prevent"]))
              }, [
                _createElementVNode("div", _hoisted_165, [
                  _createElementVNode("div", null, [
                    _createElementVNode("strong", null, _toDisplayString(_ctx.t(_ctx.modelRouteForm.editingId ? 'modelRouting.editTitle' : 'modelRouting.createTitle')), 1 /* TEXT */),
                    _createElementVNode("small", null, _toDisplayString(_ctx.t('modelRouting.orderRule')), 1 /* TEXT */)
                  ]),
                  _createElementVNode("label", _hoisted_166, [
                    _withDirectives(_createElementVNode("input", {
                      "onUpdate:modelValue": _cache[55] || (_cache[55] = $event => ((_ctx.modelRouteForm.enabled) = $event)),
                      type: "checkbox"
                    }, null, 512 /* NEED_PATCH */), [
                      [_vModelCheckbox, _ctx.modelRouteForm.enabled]
                    ]),
                    _createTextVNode(" " + _toDisplayString(_ctx.t('status.enabled')), 1 /* TEXT */)
                  ])
                ]),
                (_ctx.formError)
                  ? (_openBlock(), _createElementBlock("div", _hoisted_167, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                  : _createCommentVNode("v-if", true),
                _createElementVNode("div", _hoisted_168, [
                  _createElementVNode("label", null, [
                    _createElementVNode("span", null, _toDisplayString(_ctx.t('modelRouting.publicModel')), 1 /* TEXT */),
                    _withDirectives(_createElementVNode("input", {
                      "onUpdate:modelValue": _cache[56] || (_cache[56] = $event => ((_ctx.modelRouteForm.id) = $event)),
                      disabled: !!_ctx.modelRouteForm.editingId,
                      maxlength: "512",
                      required: ""
                    }, null, 8 /* PROPS */, _hoisted_169), [
                      [
                        _vModelText,
                        _ctx.modelRouteForm.id,
                        void 0,
                        { trim: true }
                      ]
                    ])
                  ]),
                  _createElementVNode("label", null, [
                    _createElementVNode("span", null, _toDisplayString(_ctx.t('table.name')), 1 /* TEXT */),
                    _withDirectives(_createElementVNode("input", {
                      "onUpdate:modelValue": _cache[57] || (_cache[57] = $event => ((_ctx.modelRouteForm.name) = $event)),
                      maxlength: "256"
                    }, null, 512 /* NEED_PATCH */), [
                      [
                        _vModelText,
                        _ctx.modelRouteForm.name,
                        void 0,
                        { trim: true }
                      ]
                    ])
                  ])
                ]),
                _createElementVNode("div", _hoisted_170, [
                  (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.modelRouteForm.models, (routeModel, modelIndex) => {
                    return (_openBlock(), _createElementBlock("div", {
                      key: modelIndex,
                      class: "route-model-row"
                    }, [
                      _createElementVNode("div", _hoisted_171, [
                        _createElementVNode("span", _hoisted_172, _toDisplayString(String(modelIndex + 1).padStart(2, '0')), 1 /* TEXT */),
                        _createElementVNode("label", null, [
                          _createElementVNode("span", null, _toDisplayString(_ctx.t('modelRouting.fallbackModel')), 1 /* TEXT */),
                          _withDirectives(_createElementVNode("input", {
                            "onUpdate:modelValue": $event => ((routeModel.name) = $event),
                            maxlength: "512",
                            required: ""
                          }, null, 8 /* PROPS */, _hoisted_173), [
                            [
                              _vModelText,
                              routeModel.name,
                              void 0,
                              { trim: true }
                            ]
                          ])
                        ]),
                        _createElementVNode("label", _hoisted_174, [
                          _createElementVNode("span", null, _toDisplayString(_ctx.t('table.priority')), 1 /* TEXT */),
                          _withDirectives(_createElementVNode("input", {
                            "onUpdate:modelValue": $event => ((routeModel.priority) = $event),
                            type: "number",
                            min: "0",
                            max: "1000"
                          }, null, 8 /* PROPS */, _hoisted_175), [
                            [
                              _vModelText,
                              routeModel.priority,
                              void 0,
                              { number: true }
                            ]
                          ])
                        ]),
                        _createElementVNode("label", _hoisted_176, [
                          _withDirectives(_createElementVNode("input", {
                            "onUpdate:modelValue": $event => ((routeModel.enabled) = $event),
                            type: "checkbox"
                          }, null, 8 /* PROPS */, _hoisted_177), [
                            [_vModelCheckbox, routeModel.enabled]
                          ]),
                          _createTextVNode(" " + _toDisplayString(_ctx.t('status.enabled')), 1 /* TEXT */)
                        ]),
                        _createElementVNode("button", {
                          class: "icon-button danger",
                          type: "button",
                          "aria-label": _ctx.t('action.removeModel'),
                          title: _ctx.t('action.removeModel'),
                          onClick: $event => (_ctx.removeRouteModel(modelIndex))
                        }, "×", 8 /* PROPS */, _hoisted_178)
                      ]),
                      _createElementVNode("div", _hoisted_179, [
                        (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(routeModel.targets, (target, targetIndex) => {
                          return (_openBlock(), _createElementBlock("div", {
                            key: targetIndex,
                            class: "route-target-row"
                          }, [
                            _cache[102] || (_cache[102] = _createElementVNode("span", {
                              class: "target-branch",
                              "aria-hidden": "true"
                            }, null, -1 /* CACHED */)),
                            _withDirectives(_createElementVNode("select", {
                              "onUpdate:modelValue": $event => ((target.providerId) = $event),
                              "aria-label": _ctx.t('table.provider'),
                              required: ""
                            }, [
                              _createElementVNode("option", _hoisted_181, _toDisplayString(_ctx.t('placeholder.selectProvider')), 1 /* TEXT */),
                              (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (provider) => {
                                return (_openBlock(), _createElementBlock("option", {
                                  key: provider.id,
                                  value: provider.id
                                }, _toDisplayString(provider.name) + " · P" + _toDisplayString(provider.priority), 9 /* TEXT, PROPS */, _hoisted_182))
                              }), 128 /* KEYED_FRAGMENT */))
                            ], 8 /* PROPS */, _hoisted_180), [
                              [_vModelSelect, target.providerId]
                            ]),
                            _withDirectives(_createElementVNode("input", {
                              "onUpdate:modelValue": $event => ((target.upstreamModel) = $event),
                              list: `route-models-${modelIndex}-${targetIndex}`,
                              "aria-label": _ctx.t('modelRouting.upstreamModel'),
                              placeholder: _ctx.t('modelRouting.upstreamModel'),
                              maxlength: "512",
                              required: ""
                            }, null, 8 /* PROPS */, _hoisted_183), [
                              [
                                _vModelText,
                                target.upstreamModel,
                                void 0,
                                { trim: true }
                              ]
                            ]),
                            _createElementVNode("datalist", {
                              id: `route-models-${modelIndex}-${targetIndex}`
                            }, [
                              (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(target.providerId), (model) => {
                                return (_openBlock(), _createElementBlock("option", {
                                  key: model,
                                  value: model
                                }, null, 8 /* PROPS */, _hoisted_185))
                              }), 128 /* KEYED_FRAGMENT */))
                            ], 8 /* PROPS */, _hoisted_184),
                            _createElementVNode("label", _hoisted_186, [
                              _withDirectives(_createElementVNode("input", {
                                "onUpdate:modelValue": $event => ((target.enabled) = $event),
                                type: "checkbox"
                              }, null, 8 /* PROPS */, _hoisted_187), [
                                [_vModelCheckbox, target.enabled]
                              ]),
                              _createTextVNode(" " + _toDisplayString(_ctx.t('status.enabled')), 1 /* TEXT */)
                            ]),
                            _createElementVNode("button", {
                              class: "icon-button ghost",
                              type: "button",
                              "aria-label": _ctx.t('action.removeTarget'),
                              title: _ctx.t('action.removeTarget'),
                              onClick: $event => (_ctx.removeRouteTarget(modelIndex, targetIndex))
                            }, "×", 8 /* PROPS */, _hoisted_188)
                          ]))
                        }), 128 /* KEYED_FRAGMENT */)),
                        _createElementVNode("button", {
                          class: "ghost add-target-button",
                          type: "button",
                          onClick: $event => (_ctx.addRouteTarget(modelIndex))
                        }, "+ " + _toDisplayString(_ctx.t('action.addTarget')), 9 /* TEXT, PROPS */, _hoisted_189)
                      ])
                    ]))
                  }), 128 /* KEYED_FRAGMENT */))
                ]),
                _createElementVNode("button", {
                  class: "ghost add-route-model-button",
                  type: "button",
                  onClick: _cache[58] || (_cache[58] = (...args) => (_ctx.addRouteModel && _ctx.addRouteModel(...args)))
                }, "+ " + _toDisplayString(_ctx.t('action.addFallbackModel')), 1 /* TEXT */),
                _createElementVNode("div", _hoisted_190, [
                  _createElementVNode("button", {
                    class: "ghost",
                    type: "button",
                    onClick: _cache[59] || (_cache[59] = (...args) => (_ctx.cancelModelRouteForm && _ctx.cancelModelRouteForm(...args)))
                  }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                  _createElementVNode("button", _hoisted_191, _toDisplayString(_ctx.t('action.saveModelRoute')), 1 /* TEXT */)
                ])
              ], 32 /* NEED_HYDRATION */))
            : _createCommentVNode("v-if", true),
          (_ctx.modelRoutes.length)
            ? (_openBlock(), _createElementBlock("div", _hoisted_192, [
                (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.modelRoutes, (route) => {
                  return (_openBlock(), _createElementBlock("article", {
                    key: route.id,
                    class: "model-route-item"
                  }, [
                    _createElementVNode("div", _hoisted_193, [
                      _createElementVNode("div", null, [
                        _createElementVNode("div", _hoisted_194, [
                          _createElementVNode("code", null, _toDisplayString(route.id), 1 /* TEXT */),
                          _createElementVNode("span", {
                            class: _normalizeClass(['pill', route.enabled ? 'ok' : 'bad'])
                          }, _toDisplayString(route.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */)
                        ]),
                        _createElementVNode("h3", null, _toDisplayString(route.name || route.id), 1 /* TEXT */)
                      ]),
                      _createElementVNode("div", _hoisted_195, [
                        _createElementVNode("button", {
                          class: "ghost",
                          type: "button",
                          onClick: $event => (_ctx.openModelRouteForm(route))
                        }, _toDisplayString(_ctx.t('action.edit')), 9 /* TEXT, PROPS */, _hoisted_196),
                        _createElementVNode("button", {
                          class: "ghost",
                          type: "button",
                          onClick: $event => (_ctx.toggleModelRoute(route))
                        }, _toDisplayString(route.enabled ? _ctx.t('action.disable') : _ctx.t('action.enable')), 9 /* TEXT, PROPS */, _hoisted_197),
                        _createElementVNode("button", {
                          class: "danger",
                          type: "button",
                          onClick: $event => (_ctx.deleteModelRoute(route))
                        }, _toDisplayString(_ctx.t('action.delete')), 9 /* TEXT, PROPS */, _hoisted_198)
                      ])
                    ]),
                    _createElementVNode("div", _hoisted_199, [
                      (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(route.models, (routeModel, modelIndex) => {
                        return (_openBlock(), _createElementBlock("div", {
                          key: routeModel.name,
                          class: _normalizeClass(['route-ladder-step', {disabled: !routeModel.enabled}])
                        }, [
                          _createElementVNode("span", _hoisted_200, _toDisplayString(String(modelIndex + 1).padStart(2, '0')), 1 /* TEXT */),
                          _createElementVNode("div", _hoisted_201, [
                            _createElementVNode("strong", null, _toDisplayString(routeModel.name), 1 /* TEXT */),
                            _createElementVNode("small", null, _toDisplayString(_ctx.t('table.priority')) + " " + _toDisplayString(routeModel.priority), 1 /* TEXT */)
                          ]),
                          _createElementVNode("div", _hoisted_202, [
                            (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(routeModel.targets, (target) => {
                              return (_openBlock(), _createElementBlock("span", {
                                key: target.providerId + ':' + target.upstreamModel,
                                class: _normalizeClass(['route-target-chip', {disabled: !target.enabled}])
                              }, [
                                _createElementVNode("strong", null, _toDisplayString(_ctx.providerDisplayName(target.providerId)), 1 /* TEXT */),
                                _createElementVNode("code", null, _toDisplayString(target.upstreamModel), 1 /* TEXT */),
                                (_ctx.modelStateFor(target.providerId, target.upstreamModel)?.cooldownUntil)
                                  ? (_openBlock(), _createElementBlock("small", _hoisted_203, _toDisplayString(_ctx.t('status.cooling')), 1 /* TEXT */))
                                  : _createCommentVNode("v-if", true)
                              ], 2 /* CLASS */))
                            }), 128 /* KEYED_FRAGMENT */))
                          ])
                        ], 2 /* CLASS */))
                      }), 128 /* KEYED_FRAGMENT */))
                    ])
                  ]))
                }), 128 /* KEYED_FRAGMENT */))
              ]))
            : (!_ctx.modelRouteFormOpen)
              ? (_openBlock(), _createElementBlock("div", _hoisted_204, [
                  _createElementVNode("strong", null, _toDisplayString(_ctx.t('modelRouting.empty')), 1 /* TEXT */),
                  _createElementVNode("small", null, _toDisplayString(_ctx.t('modelRouting.emptyCopy')), 1 /* TEXT */),
                  _createElementVNode("button", {
                    type: "button",
                    onClick: _cache[61] || (_cache[61] = $event => (_ctx.openModelRouteForm()))
                  }, _toDisplayString(_ctx.t('action.addModelRoute')), 1 /* TEXT */)
                ]))
              : _createCommentVNode("v-if", true)
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'model-routing']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_205, [
        _createElementVNode("article", _hoisted_206, [
          _createElementVNode("div", _hoisted_207, [
            _createElementVNode("strong", null, _toDisplayString(_ctx.t('keys.title')), 1 /* TEXT */),
            _createElementVNode("span", null, _toDisplayString(_ctx.t('keys.copy')), 1 /* TEXT */)
          ]),
          _createElementVNode("form", {
            class: "form-grid gateway-key-form",
            onSubmit: _cache[63] || (_cache[63] = _withModifiers((...args) => (_ctx.createGatewayKey && _ctx.createGatewayKey(...args)), ["prevent"]))
          }, [
            _withDirectives(_createElementVNode("input", {
              "onUpdate:modelValue": _cache[62] || (_cache[62] = $event => ((_ctx.gatewayKeyForm.name) = $event)),
              placeholder: _ctx.t('placeholder.gatewayKeyName')
            }, null, 8 /* PROPS */, _hoisted_208), [
              [
                _vModelText,
                _ctx.gatewayKeyForm.name,
                void 0,
                { trim: true }
              ]
            ]),
            _createElementVNode("div", _hoisted_209, [
              _createElementVNode("button", _hoisted_210, _toDisplayString(_ctx.t('action.addGatewayKey')), 1 /* TEXT */)
            ])
          ], 32 /* NEED_HYDRATION */),
          (_ctx.createdGatewayKey?.plaintext)
            ? (_openBlock(), _createElementBlock("div", _hoisted_211, [
                _createElementVNode("strong", null, _toDisplayString(_ctx.t(_ctx.createdGatewayKey.event === 'rotated' ? 'gateway.rotatedTitle' : 'gateway.createdTitle')), 1 /* TEXT */),
                _createElementVNode("span", null, _toDisplayString(_ctx.t(_ctx.createdGatewayKey.event === 'rotated' ? 'gateway.rotatedCopy' : 'gateway.createdCopy')), 1 /* TEXT */),
                _createElementVNode("code", _hoisted_212, _toDisplayString(_ctx.createdGatewayKey.plaintext), 1 /* TEXT */),
                _createElementVNode("button", {
                  type: "button",
                  class: _normalizeClass(_ctx.copyButtonClass('createdGatewayKey')),
                  onClick: _cache[64] || (_cache[64] = $event => (_ctx.copyText(_ctx.createdGatewayKey.plaintext, 'createdGatewayKey')))
                }, _toDisplayString(_ctx.copyButtonText('createdGatewayKey')), 3 /* TEXT, CLASS */)
              ]))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_213, [
            _createElementVNode("strong", null, _toDisplayString(_ctx.t('gateway.compatTitle')), 1 /* TEXT */),
            _createElementVNode("span", null, _toDisplayString(_ctx.t('gateway.compatCopy')), 1 /* TEXT */)
          ]),
          (!_ctx.gatewayKeys.length)
            ? (_openBlock(), _createElementBlock("div", _hoisted_214, [
                _createElementVNode("strong", null, _toDisplayString(_ctx.t('nav.keys')), 1 /* TEXT */),
                _createElementVNode("small", null, _toDisplayString(_ctx.t('keys.copy')), 1 /* TEXT */)
              ]))
            : (_openBlock(), _createElementBlock("div", _hoisted_215, [
                _createElementVNode("table", null, [
                  _createElementVNode("thead", null, [
                    _createElementVNode("tr", null, [
                      _cache[103] || (_cache[103] = _createElementVNode("th", null, "ID", -1 /* CACHED */)),
                      _createElementVNode("th", null, _toDisplayString(_ctx.t('table.name')), 1 /* TEXT */),
                      _cache[104] || (_cache[104] = _createElementVNode("th", null, "Key", -1 /* CACHED */)),
                      _createElementVNode("th", null, _toDisplayString(_ctx.t('table.requests')), 1 /* TEXT */),
                      _createElementVNode("th", null, _toDisplayString(_ctx.t('table.lastUsed')), 1 /* TEXT */),
                      _createElementVNode("th", null, _toDisplayString(_ctx.t('table.status')), 1 /* TEXT */),
                      _createElementVNode("th", null, _toDisplayString(_ctx.t('table.actions')), 1 /* TEXT */)
                    ])
                  ]),
                  _createElementVNode("tbody", null, [
                    (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.gatewayKeys, (k) => {
                      return (_openBlock(), _createElementBlock("tr", {
                        key: k.id
                      }, [
                        _createElementVNode("td", null, [
                          _createElementVNode("code", null, _toDisplayString(k.id), 1 /* TEXT */)
                        ]),
                        _createElementVNode("td", null, _toDisplayString(k.name), 1 /* TEXT */),
                        _createElementVNode("td", null, [
                          _createElementVNode("code", _hoisted_216, _toDisplayString(k.keyHint), 1 /* TEXT */)
                        ]),
                        _createElementVNode("td", null, _toDisplayString(k.requestCount), 1 /* TEXT */),
                        _createElementVNode("td", null, _toDisplayString(_ctx.formatGatewayTime(k.lastUsedAt)), 1 /* TEXT */),
                        _createElementVNode("td", null, [
                          _createElementVNode("span", {
                            class: _normalizeClass(['pill', k.enabled ? 'ok' : 'bad'])
                          }, _toDisplayString(k.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */)
                        ]),
                        _createElementVNode("td", _hoisted_217, [
                          _createElementVNode("button", {
                            class: "ghost",
                            type: "button",
                            onClick: $event => (_ctx.toggleGatewayKey(k))
                          }, _toDisplayString(k.enabled ? _ctx.t('action.disable') : _ctx.t('action.enable')), 9 /* TEXT, PROPS */, _hoisted_218),
                          _createElementVNode("button", {
                            class: "ghost",
                            type: "button",
                            onClick: $event => (_ctx.rotateGatewayKey(k))
                          }, _toDisplayString(_ctx.t('action.rotate')), 9 /* TEXT, PROPS */, _hoisted_219),
                          _createElementVNode("button", {
                            class: "danger",
                            type: "button",
                            onClick: $event => (_ctx.deleteGatewayKey(k))
                          }, _toDisplayString(_ctx.t('action.delete')), 9 /* TEXT, PROPS */, _hoisted_220)
                        ])
                      ]))
                    }), 128 /* KEYED_FRAGMENT */))
                  ])
                ])
              ]))
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'keys']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_221, [
        _createElementVNode("article", _hoisted_222, [
          _createElementVNode("form", {
            class: "form-grid routing-form",
            onSubmit: _cache[80] || (_cache[80] = _withModifiers((...args) => (_ctx.saveRouting && _ctx.saveRouting(...args)), ["prevent"]))
          }, [
            _createElementVNode("div", _hoisted_223, [
              _createElementVNode("div", _hoisted_224, [
                _createElementVNode("div", null, [
                  _createElementVNode("strong", null, _toDisplayString(_ctx.t('routing.simpleTitle')), 1 /* TEXT */),
                  _createElementVNode("small", null, _toDisplayString(_ctx.t('routing.simpleCopy')), 1 /* TEXT */)
                ])
              ]),
              _createElementVNode("div", _hoisted_225, [
                _createElementVNode("label", null, [
                  _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.failureThreshold')), 1 /* TEXT */),
                  _withDirectives(_createElementVNode("input", {
                    "onUpdate:modelValue": _cache[65] || (_cache[65] = $event => ((_ctx.routing.failureThreshold) = $event)),
                    type: "number",
                    min: "1"
                  }, null, 512 /* NEED_PATCH */), [
                    [
                      _vModelText,
                      _ctx.routing.failureThreshold,
                      void 0,
                      { number: true }
                    ]
                  ])
                ]),
                _createElementVNode("label", null, [
                  _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.cooldownSeconds')), 1 /* TEXT */),
                  _withDirectives(_createElementVNode("input", {
                    "onUpdate:modelValue": _cache[66] || (_cache[66] = $event => ((_ctx.routing.cooldownSeconds) = $event)),
                    type: "number",
                    min: "0"
                  }, null, 512 /* NEED_PATCH */), [
                    [
                      _vModelText,
                      _ctx.routing.cooldownSeconds,
                      void 0,
                      { number: true }
                    ]
                  ])
                ]),
                _createElementVNode("label", null, [
                  _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.retryPerRequest')), 1 /* TEXT */),
                  _withDirectives(_createElementVNode("input", {
                    "onUpdate:modelValue": _cache[67] || (_cache[67] = $event => ((_ctx.routing.retryPerRequest) = $event)),
                    type: "number",
                    min: "1"
                  }, null, 512 /* NEED_PATCH */), [
                    [
                      _vModelText,
                      _ctx.routing.retryPerRequest,
                      void 0,
                      { number: true }
                    ]
                  ])
                ]),
                _createElementVNode("label", null, [
                  _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.timeoutSeconds')), 1 /* TEXT */),
                  _withDirectives(_createElementVNode("input", {
                    "onUpdate:modelValue": _cache[68] || (_cache[68] = $event => ((_ctx.routing.timeoutSeconds) = $event)),
                    type: "number",
                    min: "1"
                  }, null, 512 /* NEED_PATCH */), [
                    [
                      _vModelText,
                      _ctx.routing.timeoutSeconds,
                      void 0,
                      { number: true }
                    ]
                  ])
                ]),
                _createElementVNode("label", null, [
                  _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.maxConcurrentRequests')), 1 /* TEXT */),
                  _withDirectives(_createElementVNode("input", {
                    "onUpdate:modelValue": _cache[69] || (_cache[69] = $event => ((_ctx.routing.maxConcurrentRequests) = $event)),
                    type: "number",
                    min: "0"
                  }, null, 512 /* NEED_PATCH */), [
                    [
                      _vModelText,
                      _ctx.routing.maxConcurrentRequests,
                      void 0,
                      { number: true }
                    ]
                  ])
                ]),
                _createElementVNode("label", _hoisted_226, [
                  _withDirectives(_createElementVNode("input", {
                    "onUpdate:modelValue": _cache[70] || (_cache[70] = $event => ((_ctx.routing.streamRetryBeforeFirstByte) = $event)),
                    type: "checkbox"
                  }, null, 512 /* NEED_PATCH */), [
                    [_vModelCheckbox, _ctx.routing.streamRetryBeforeFirstByte]
                  ]),
                  _createTextVNode(" " + _toDisplayString(_ctx.t('routing.streamRetryBeforeFirstByte')), 1 /* TEXT */)
                ])
              ])
            ]),
            _createElementVNode("div", _hoisted_227, [
              _createElementVNode("button", {
                class: "ghost",
                type: "button",
                "aria-expanded": _ctx.routingAdvanced,
                "aria-controls": "routing-advanced",
                onClick: _cache[71] || (_cache[71] = $event => (_ctx.routingAdvanced = !_ctx.routingAdvanced))
              }, _toDisplayString(_ctx.routingAdvanced ? _ctx.t('routing.hideAdvanced') : _ctx.t('routing.showAdvanced')), 9 /* TEXT, PROPS */, _hoisted_228)
            ]),
            (_ctx.routingAdvanced)
              ? (_openBlock(), _createElementBlock("div", _hoisted_229, [
                  _createElementVNode("div", _hoisted_230, [
                    _createElementVNode("div", null, [
                      _createElementVNode("strong", null, _toDisplayString(_ctx.t('routing.advancedTitle')), 1 /* TEXT */),
                      _createElementVNode("small", null, _toDisplayString(_ctx.t('routing.advancedCopy')), 1 /* TEXT */)
                    ])
                  ]),
                  _createElementVNode("div", _hoisted_231, [
                    _createElementVNode("label", null, [
                      _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.authCooldownSeconds')), 1 /* TEXT */),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[72] || (_cache[72] = $event => ((_ctx.routing.authCooldownSeconds) = $event)),
                        type: "number",
                        min: "0"
                      }, null, 512 /* NEED_PATCH */), [
                        [
                          _vModelText,
                          _ctx.routing.authCooldownSeconds,
                          void 0,
                          { number: true }
                        ]
                      ])
                    ]),
                    _createElementVNode("label", null, [
                      _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.streamIdleTimeoutSeconds')), 1 /* TEXT */),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[73] || (_cache[73] = $event => ((_ctx.routing.streamIdleTimeoutSeconds) = $event)),
                        type: "number",
                        min: "1"
                      }, null, 512 /* NEED_PATCH */), [
                        [
                          _vModelText,
                          _ctx.routing.streamIdleTimeoutSeconds,
                          void 0,
                          { number: true }
                        ]
                      ])
                    ]),
                    _createElementVNode("label", null, [
                      _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.streamWriteTimeoutSeconds')), 1 /* TEXT */),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[74] || (_cache[74] = $event => ((_ctx.routing.streamWriteTimeoutSeconds) = $event)),
                        type: "number",
                        min: "1"
                      }, null, 512 /* NEED_PATCH */), [
                        [
                          _vModelText,
                          _ctx.routing.streamWriteTimeoutSeconds,
                          void 0,
                          { number: true }
                        ]
                      ])
                    ]),
                    _createElementVNode("label", null, [
                      _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.maxConcurrentPerProvider')), 1 /* TEXT */),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[75] || (_cache[75] = $event => ((_ctx.routing.maxConcurrentPerProvider) = $event)),
                        type: "number",
                        min: "0"
                      }, null, 512 /* NEED_PATCH */), [
                        [
                          _vModelText,
                          _ctx.routing.maxConcurrentPerProvider,
                          void 0,
                          { number: true }
                        ]
                      ])
                    ]),
                    _createElementVNode("label", null, [
                      _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.maxConcurrentPerKey')), 1 /* TEXT */),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[76] || (_cache[76] = $event => ((_ctx.routing.maxConcurrentPerKey) = $event)),
                        type: "number",
                        min: "0"
                      }, null, 512 /* NEED_PATCH */), [
                        [
                          _vModelText,
                          _ctx.routing.maxConcurrentPerKey,
                          void 0,
                          { number: true }
                        ]
                      ])
                    ]),
                    _createElementVNode("label", null, [
                      _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.queueTimeoutMilliseconds')), 1 /* TEXT */),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[77] || (_cache[77] = $event => ((_ctx.routing.queueTimeoutMilliseconds) = $event)),
                        type: "number",
                        min: "0"
                      }, null, 512 /* NEED_PATCH */), [
                        [
                          _vModelText,
                          _ctx.routing.queueTimeoutMilliseconds,
                          void 0,
                          { number: true }
                        ]
                      ])
                    ])
                  ]),
                  _createElementVNode("div", _hoisted_232, [
                    _createElementVNode("strong", null, _toDisplayString(_ctx.t('routing.riskTitle')), 1 /* TEXT */),
                    _createElementVNode("small", null, _toDisplayString(_ctx.t('routing.riskCopy')), 1 /* TEXT */),
                    _createElementVNode("label", _hoisted_233, [
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[78] || (_cache[78] = $event => ((_ctx.routing.retryAmbiguousErrors) = $event)),
                        type: "checkbox"
                      }, null, 512 /* NEED_PATCH */), [
                        [_vModelCheckbox, _ctx.routing.retryAmbiguousErrors]
                      ]),
                      _createTextVNode(" " + _toDisplayString(_ctx.t('routing.retryAmbiguousErrors')), 1 /* TEXT */)
                    ]),
                    _createElementVNode("label", _hoisted_234, [
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[79] || (_cache[79] = $event => ((_ctx.routing.allowInsecureUpstreams) = $event)),
                        type: "checkbox"
                      }, null, 512 /* NEED_PATCH */), [
                        [_vModelCheckbox, _ctx.routing.allowInsecureUpstreams]
                      ]),
                      _createTextVNode(" " + _toDisplayString(_ctx.t('routing.allowInsecureUpstreams')), 1 /* TEXT */)
                    ])
                  ])
                ]))
              : _createCommentVNode("v-if", true),
            _createElementVNode("div", _hoisted_235, [
              _createElementVNode("button", _hoisted_236, _toDisplayString(_ctx.t('action.save')), 1 /* TEXT */)
            ])
          ], 32 /* NEED_HYDRATION */),
          (_ctx.routingMessage)
            ? (_openBlock(), _createElementBlock("div", _hoisted_237, _toDisplayString(_ctx.routingMessage), 1 /* TEXT */))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_238, [
            _createElementVNode("h2", null, _toDisplayString(_ctx.t('maintenance.title')), 1 /* TEXT */),
            _createElementVNode("div", _hoisted_239, [
              _createElementVNode("button", {
                class: "ghost",
                type: "button",
                onClick: _cache[81] || (_cache[81] = (...args) => (_ctx.checkIntegrity && _ctx.checkIntegrity(...args)))
              }, _toDisplayString(_ctx.t('maintenance.checkIntegrity')), 1 /* TEXT */),
              _createElementVNode("button", {
                class: "ghost",
                type: "button",
                onClick: _cache[82] || (_cache[82] = (...args) => (_ctx.downloadBackup && _ctx.downloadBackup(...args)))
              }, _toDisplayString(_ctx.t('maintenance.downloadDatabaseBackup')), 1 /* TEXT */)
            ])
          ]),
          _createElementVNode("form", {
            class: "portable-backup-form",
            onSubmit: _cache[85] || (_cache[85] = _withModifiers((...args) => (_ctx.downloadPortableBackup && _ctx.downloadPortableBackup(...args)), ["prevent"]))
          }, [
            _createElementVNode("label", null, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('maintenance.passphrase')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[83] || (_cache[83] = $event => ((_ctx.backupPassphrase) = $event)),
                type: "password",
                minlength: "12",
                autocomplete: "new-password",
                required: ""
              }, null, 512 /* NEED_PATCH */), [
                [_vModelText, _ctx.backupPassphrase]
              ])
            ]),
            _createElementVNode("label", null, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('maintenance.confirmPassphrase')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[84] || (_cache[84] = $event => ((_ctx.backupPassphraseConfirm) = $event)),
                type: "password",
                minlength: "12",
                autocomplete: "new-password",
                required: ""
              }, null, 512 /* NEED_PATCH */), [
                [_vModelText, _ctx.backupPassphraseConfirm]
              ])
            ]),
            _createElementVNode("button", _hoisted_240, _toDisplayString(_ctx.t('maintenance.downloadPortableBackup')), 1 /* TEXT */)
          ], 32 /* NEED_HYDRATION */),
          (_ctx.maintenanceMessage)
            ? (_openBlock(), _createElementBlock("div", _hoisted_241, _toDisplayString(_ctx.maintenanceMessage), 1 /* TEXT */))
            : _createCommentVNode("v-if", true)
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'routing']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_242, [
        _createElementVNode("article", _hoisted_243, [
          _createElementVNode("div", _hoisted_244, [
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              onClick: _cache[86] || (_cache[86] = $event => (_ctx.setActiveSection('providers')))
            }, _toDisplayString(_ctx.t('action.backProviders')), 1 /* TEXT */)
          ]),
          _createElementVNode("div", _hoisted_245, [
            _createElementVNode("strong", null, _toDisplayString(_ctx.t('setup.title')), 1 /* TEXT */),
            _createElementVNode("span", null, _toDisplayString(_ctx.t('setup.copy')), 1 /* TEXT */)
          ]),
          _createElementVNode("div", _hoisted_246, [
            (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.snippets, (value, name) => {
              return (_openBlock(), _createElementBlock("div", {
                key: name,
                class: "snippet"
              }, [
                _createElementVNode("div", _hoisted_247, [
                  _createElementVNode("h3", null, _toDisplayString(name), 1 /* TEXT */),
                  _createElementVNode("button", {
                    class: _normalizeClass(["ghost copy-chip", _ctx.copyButtonClass(`snippet:${name}`)]),
                    type: "button",
                    onClick: $event => (_ctx.copyText(typeof value === 'string' ? value : _ctx.pretty(value), `snippet:${name}`))
                  }, _toDisplayString(_ctx.copyButtonText(`snippet:${name}`)), 11 /* TEXT, CLASS, PROPS */, _hoisted_248)
                ]),
                _createElementVNode("pre", null, _toDisplayString(typeof value === 'string' ? value : _ctx.pretty(value)), 1 /* TEXT */)
              ]))
            }), 128 /* KEYED_FRAGMENT */))
          ])
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'setup']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_249, [
        _createElementVNode("article", _hoisted_250, [
          _createElementVNode("div", _hoisted_251, [
            _withDirectives(_createElementVNode("input", {
              "onUpdate:modelValue": _cache[87] || (_cache[87] = $event => ((_ctx.logFilters.q) = $event)),
              "aria-label": _ctx.t('logs.search'),
              placeholder: _ctx.t('logs.search'),
              onKeyup: _cache[88] || (_cache[88] = _withKeys($event => (_ctx.loadLogs(0)), ["enter"]))
            }, null, 40 /* PROPS, NEED_HYDRATION */, _hoisted_252), [
              [
                _vModelText,
                _ctx.logFilters.q,
                void 0,
                { trim: true }
              ]
            ]),
            _withDirectives(_createElementVNode("select", {
              "onUpdate:modelValue": _cache[89] || (_cache[89] = $event => ((_ctx.logFilters.status) = $event)),
              "aria-label": _ctx.t('table.status')
            }, [
              _createElementVNode("option", _hoisted_254, _toDisplayString(_ctx.t('logs.allStatus')), 1 /* TEXT */),
              _cache[105] || (_cache[105] = _createStaticVNode("<option value=\"200\">2xx</option><option value=\"400\">400</option><option value=\"401\">401</option><option value=\"429\">429</option><option value=\"499\">499</option><option value=\"500\">500</option><option value=\"502\">502</option><option value=\"503\">503</option>", 8))
            ], 8 /* PROPS */, _hoisted_253), [
              [_vModelSelect, _ctx.logFilters.status]
            ]),
            _withDirectives(_createElementVNode("select", {
              "onUpdate:modelValue": _cache[90] || (_cache[90] = $event => ((_ctx.logFilters.providerId) = $event)),
              "aria-label": _ctx.t('table.provider')
            }, [
              _createElementVNode("option", _hoisted_256, _toDisplayString(_ctx.t('logs.allProviders')), 1 /* TEXT */),
              (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (provider) => {
                return (_openBlock(), _createElementBlock("option", {
                  key: provider.id,
                  value: provider.id
                }, _toDisplayString(provider.name), 9 /* TEXT, PROPS */, _hoisted_257))
              }), 128 /* KEYED_FRAGMENT */))
            ], 8 /* PROPS */, _hoisted_255), [
              [_vModelSelect, _ctx.logFilters.providerId]
            ]),
            _withDirectives(_createElementVNode("input", {
              "onUpdate:modelValue": _cache[91] || (_cache[91] = $event => ((_ctx.logFilters.model) = $event)),
              "aria-label": _ctx.t('table.model'),
              placeholder: _ctx.t('table.model'),
              onKeyup: _cache[92] || (_cache[92] = _withKeys($event => (_ctx.loadLogs(0)), ["enter"]))
            }, null, 40 /* PROPS, NEED_HYDRATION */, _hoisted_258), [
              [
                _vModelText,
                _ctx.logFilters.model,
                void 0,
                { trim: true }
              ]
            ]),
            _createElementVNode("button", {
              type: "button",
              disabled: _ctx.logsLoading,
              onClick: _cache[93] || (_cache[93] = $event => (_ctx.loadLogs(0)))
            }, _toDisplayString(_ctx.t('logs.apply')), 9 /* TEXT, PROPS */, _hoisted_259),
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              onClick: _cache[94] || (_cache[94] = (...args) => (_ctx.exportLogs && _ctx.exportLogs(...args)))
            }, _toDisplayString(_ctx.t('logs.export')), 1 /* TEXT */),
            _createElementVNode("button", {
              class: "danger",
              type: "button",
              onClick: _cache[95] || (_cache[95] = (...args) => (_ctx.clearLogs && _ctx.clearLogs(...args)))
            }, _toDisplayString(_ctx.t('logs.clear')), 1 /* TEXT */)
          ]),
          _createElementVNode("div", _hoisted_260, [
            _createElementVNode("table", null, [
              _createElementVNode("thead", null, [
                _createElementVNode("tr", null, [
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.time')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.protocol')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.model')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.provider')), 1 /* TEXT */),
                  _cache[106] || (_cache[106] = _createElementVNode("th", null, "Key", -1 /* CACHED */)),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.status')), 1 /* TEXT */),
                  _cache[107] || (_cache[107] = _createElementVNode("th", null, "Tokens", -1 /* CACHED */)),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.latency')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.error')), 1 /* TEXT */)
                ])
              ]),
              _createElementVNode("tbody", null, [
                (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.logs, (log) => {
                  return (_openBlock(), _createElementBlock("tr", {
                    key: log.id
                  }, [
                    _createElementVNode("td", null, _toDisplayString(_ctx.formatGatewayTime(log.createdAt)), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.inboundProtocol), 1 /* TEXT */),
                    _createElementVNode("td", _hoisted_261, [
                      _createElementVNode("code", null, _toDisplayString(log.model || '-'), 1 /* TEXT */),
                      (log.upstreamModel && log.upstreamModel !== log.model)
                        ? (_openBlock(), _createElementBlock("small", _hoisted_262, _toDisplayString(log.upstreamModel), 1 /* TEXT */))
                        : _createCommentVNode("v-if", true),
                      (log.routeId)
                        ? (_openBlock(), _createElementBlock("small", _hoisted_263, _toDisplayString(log.routeId) + " · " + _toDisplayString(log.attempts || 1) + " " + _toDisplayString(_ctx.t('logs.attempts')), 1 /* TEXT */))
                        : _createCommentVNode("v-if", true)
                    ]),
                    _createElementVNode("td", null, _toDisplayString(log.providerId || '-'), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.keyId || '-'), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.status), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.totalTokens || '-'), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.latencyMs) + "ms", 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.errorType || '-'), 1 /* TEXT */)
                  ]))
                }), 128 /* KEYED_FRAGMENT */))
              ])
            ])
          ]),
          _createElementVNode("div", _hoisted_264, [
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              disabled: _ctx.logPage.offset <= 0 || _ctx.logsLoading,
              onClick: _cache[96] || (_cache[96] = $event => (_ctx.loadLogs(_ctx.logPage.offset - _ctx.logPage.limit)))
            }, _toDisplayString(_ctx.t('logs.previous')), 9 /* TEXT, PROPS */, _hoisted_265),
            _createElementVNode("span", null, _toDisplayString(_ctx.logPage.total ? `${_ctx.logPage.offset + 1}-${Math.min(_ctx.logPage.offset + _ctx.logs.length, _ctx.logPage.total)} / ${_ctx.logPage.total}` : '0 / 0'), 1 /* TEXT */),
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              disabled: _ctx.logPage.offset + _ctx.logPage.limit >= _ctx.logPage.total || _ctx.logsLoading,
              onClick: _cache[97] || (_cache[97] = $event => (_ctx.loadLogs(_ctx.logPage.offset + _ctx.logPage.limit)))
            }, _toDisplayString(_ctx.t('logs.next')), 9 /* TEXT, PROPS */, _hoisted_266)
          ])
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'logs']
      ])
    ])
  ], 64 /* STABLE_FRAGMENT */))
}
