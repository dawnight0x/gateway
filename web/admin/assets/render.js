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
const _hoisted_25 = { class: "panel-title" }
const _hoisted_26 = { class: "log-list" }
const _hoisted_27 = {
  key: 0,
  class: "empty-state"
}
const _hoisted_28 = { class: "section" }
const _hoisted_29 = { class: "panel" }
const _hoisted_30 = { class: "panel-title section-actions" }
const _hoisted_31 = { class: "panel-actions" }
const _hoisted_32 = ["disabled"]
const _hoisted_33 = { class: "provider-summary-grid" }
const _hoisted_34 = { class: "summary-card" }
const _hoisted_35 = { class: "summary-card" }
const _hoisted_36 = { class: "summary-card" }
const _hoisted_37 = {
  key: 0,
  class: "form-error"
}
const _hoisted_38 = ["disabled", "placeholder"]
const _hoisted_39 = ["placeholder"]
const _hoisted_40 = ["value"]
const _hoisted_41 = ["placeholder"]
const _hoisted_42 = ["placeholder"]
const _hoisted_43 = ["placeholder"]
const _hoisted_44 = { class: "model-select-field" }
const _hoisted_45 = { value: "" }
const _hoisted_46 = ["value"]
const _hoisted_47 = { class: "form-section" }
const _hoisted_48 = ["placeholder"]
const _hoisted_49 = ["placeholder"]
const _hoisted_50 = ["placeholder"]
const _hoisted_51 = { class: "advanced-field" }
const _hoisted_52 = ["placeholder"]
const _hoisted_53 = { class: "quick-fill" }
const _hoisted_54 = { class: "form-actions" }
const _hoisted_55 = { type: "submit" }
const _hoisted_56 = { class: "provider-board" }
const _hoisted_57 = { class: "provider-card-main" }
const _hoisted_58 = { class: "provider-title-row" }
const _hoisted_59 = { class: "pill" }
const _hoisted_60 = { class: "provider-endpoint" }
const _hoisted_61 = { class: "provider-actions" }
const _hoisted_62 = ["onClick"]
const _hoisted_63 = ["onClick"]
const _hoisted_64 = ["onClick"]
const _hoisted_65 = ["onClick"]
const _hoisted_66 = { class: "inline-form-heading" }
const _hoisted_67 = {
  key: 0,
  class: "form-error"
}
const _hoisted_68 = ["placeholder"]
const _hoisted_69 = ["placeholder"]
const _hoisted_70 = ["value"]
const _hoisted_71 = ["placeholder"]
const _hoisted_72 = ["placeholder"]
const _hoisted_73 = ["placeholder"]
const _hoisted_74 = { class: "model-select-field" }
const _hoisted_75 = { value: "" }
const _hoisted_76 = ["value"]
const _hoisted_77 = { class: "advanced-field" }
const _hoisted_78 = ["placeholder"]
const _hoisted_79 = { class: "quick-fill" }
const _hoisted_80 = { class: "form-actions" }
const _hoisted_81 = { type: "submit" }
const _hoisted_82 = { class: "provider-meta-grid" }
const _hoisted_83 = { class: "inline-form-heading" }
const _hoisted_84 = {
  key: 0,
  class: "form-error"
}
const _hoisted_85 = ["value"]
const _hoisted_86 = ["placeholder"]
const _hoisted_87 = ["placeholder"]
const _hoisted_88 = ["placeholder"]
const _hoisted_89 = { class: "model-select-field" }
const _hoisted_90 = { value: "" }
const _hoisted_91 = ["value"]
const _hoisted_92 = { class: "check" }
const _hoisted_93 = { class: "form-actions" }
const _hoisted_94 = { type: "submit" }
const _hoisted_95 = {
  key: 2,
  class: "key-stack"
}
const _hoisted_96 = { class: "key-card-main" }
const _hoisted_97 = { class: "key-title-row" }
const _hoisted_98 = { class: "key-name" }
const _hoisted_99 = {
  key: 0,
  class: "pill ok"
}
const _hoisted_100 = { class: "key-hint" }
const _hoisted_101 = { class: "inline-form-heading" }
const _hoisted_102 = {
  key: 0,
  class: "form-error"
}
const _hoisted_103 = ["value"]
const _hoisted_104 = ["placeholder"]
const _hoisted_105 = ["placeholder"]
const _hoisted_106 = ["placeholder"]
const _hoisted_107 = { class: "model-select-field" }
const _hoisted_108 = { value: "" }
const _hoisted_109 = ["value"]
const _hoisted_110 = { class: "check" }
const _hoisted_111 = { class: "form-actions" }
const _hoisted_112 = { type: "submit" }
const _hoisted_113 = { class: "key-test-row" }
const _hoisted_114 = ["onClick", "disabled"]
const _hoisted_115 = {
  key: 2,
  class: "test-error"
}
const _hoisted_116 = { class: "actions key-actions" }
const _hoisted_117 = ["onClick"]
const _hoisted_118 = ["onClick"]
const _hoisted_119 = ["onClick"]
const _hoisted_120 = ["onClick"]
const _hoisted_121 = ["onClick"]
const _hoisted_122 = {
  key: 3,
  class: "empty-state compact-empty provider-empty"
}
const _hoisted_123 = ["onClick"]
const _hoisted_124 = {
  key: 0,
  class: "empty-state compact-empty"
}
const _hoisted_125 = { class: "balance-header" }
const _hoisted_126 = ["disabled"]
const _hoisted_127 = {
  key: 1,
  class: "compact-list"
}
const _hoisted_128 = { class: "table-wrap" }
const _hoisted_129 = { key: 0 }
const _hoisted_130 = { colspan: "6" }
const _hoisted_131 = { class: "empty-state compact-empty" }
const _hoisted_132 = { class: "section" }
const _hoisted_133 = { class: "panel" }
const _hoisted_134 = { class: "flow-hint" }
const _hoisted_135 = ["placeholder"]
const _hoisted_136 = { class: "form-actions" }
const _hoisted_137 = { type: "submit" }
const _hoisted_138 = {
  key: 0,
  class: "flow-hint"
}
const _hoisted_139 = { class: "key-hint" }
const _hoisted_140 = { class: "flow-hint" }
const _hoisted_141 = {
  key: 1,
  class: "empty-state compact-empty"
}
const _hoisted_142 = {
  key: 2,
  class: "table-wrap"
}
const _hoisted_143 = { class: "key-hint" }
const _hoisted_144 = { class: "actions" }
const _hoisted_145 = ["onClick"]
const _hoisted_146 = ["onClick"]
const _hoisted_147 = ["onClick"]
const _hoisted_148 = { class: "section" }
const _hoisted_149 = { class: "panel" }
const _hoisted_150 = { class: "check" }
const _hoisted_151 = { class: "check" }
const _hoisted_152 = { class: "check" }
const _hoisted_153 = { class: "form-actions" }
const _hoisted_154 = { type: "submit" }
const _hoisted_155 = {
  key: 0,
  class: "form-success",
  role: "status"
}
const _hoisted_156 = { class: "panel-title maintenance-actions" }
const _hoisted_157 = { class: "panel-actions" }
const _hoisted_158 = { type: "submit" }
const _hoisted_159 = {
  key: 1,
  class: "form-success",
  role: "status"
}
const _hoisted_160 = { class: "section" }
const _hoisted_161 = { class: "panel" }
const _hoisted_162 = { class: "panel-title section-actions" }
const _hoisted_163 = { class: "flow-hint" }
const _hoisted_164 = { class: "snippet-grid" }
const _hoisted_165 = { class: "snippet-head" }
const _hoisted_166 = ["onClick"]
const _hoisted_167 = { class: "section" }
const _hoisted_168 = { class: "panel" }
const _hoisted_169 = { class: "log-toolbar" }
const _hoisted_170 = ["aria-label", "placeholder"]
const _hoisted_171 = ["aria-label"]
const _hoisted_172 = { value: "" }
const _hoisted_173 = ["aria-label"]
const _hoisted_174 = { value: "" }
const _hoisted_175 = ["value"]
const _hoisted_176 = ["aria-label", "placeholder"]
const _hoisted_177 = ["disabled"]
const _hoisted_178 = { class: "table-wrap" }
const _hoisted_179 = { class: "pagination" }
const _hoisted_180 = ["disabled"]
const _hoisted_181 = ["disabled"]

window.gatewayRender = function render(_ctx, _cache) {
  return (_openBlock(), _createElementBlock(_Fragment, null, [
    _createElementVNode("aside", _hoisted_1, [
      _createElementVNode("div", _hoisted_2, [
        _cache[90] || (_cache[90] = _createElementVNode("img", {
          class: "brand-mark",
          src: "/admin/assets/app-icon.svg",
          alt: ""
        }, null, -1 /* CACHED */)),
        _createElementVNode("div", null, [
          _cache[89] || (_cache[89] = _createElementVNode("strong", null, "Local AI Gateway", -1 /* CACHED */)),
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
          _cache[91] || (_cache[91] = _createElementVNode("div", { class: "protocol-strip" }, [
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
              _createElementVNode("h2", null, _toDisplayString(_ctx.t('dashboard.recent')), 1 /* TEXT */)
            ]),
            _createElementVNode("div", _hoisted_26, [
              (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.recentLogs, (log) => {
                return (_openBlock(), _createElementBlock("div", {
                  key: log.id,
                  class: "log-row"
                }, [
                  _createElementVNode("span", null, _toDisplayString(log.status), 1 /* TEXT */),
                  _createElementVNode("strong", null, _toDisplayString(log.model || '-'), 1 /* TEXT */),
                  _createElementVNode("small", null, _toDisplayString(log.inboundProtocol) + " · " + _toDisplayString(log.latencyMs) + "ms · " + _toDisplayString(log.errorType || 'ok'), 1 /* TEXT */)
                ]))
              }), 128 /* KEYED_FRAGMENT */)),
              (!_ctx.recentLogs.length)
                ? (_openBlock(), _createElementBlock("div", _hoisted_27, [
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
      _withDirectives(_createElementVNode("section", _hoisted_28, [
        _createElementVNode("article", _hoisted_29, [
          _createElementVNode("div", _hoisted_30, [
            _createElementVNode("div", _hoisted_31, [
              _createElementVNode("button", {
                class: "ghost",
                type: "button",
                onClick: _cache[7] || (_cache[7] = (...args) => (_ctx.refreshBalances && _ctx.refreshBalances(...args))),
                disabled: _ctx.balanceRefreshing
              }, _toDisplayString(_ctx.balanceRefreshing ? _ctx.t('action.refreshingBalance') : _ctx.t('action.refreshBalance')), 9 /* TEXT, PROPS */, _hoisted_32),
              _createElementVNode("button", {
                type: "button",
                onClick: _cache[8] || (_cache[8] = (...args) => (_ctx.openProviderForm && _ctx.openProviderForm(...args)))
              }, _toDisplayString(_ctx.t('action.addProvider')), 1 /* TEXT */)
            ])
          ]),
          _createElementVNode("div", _hoisted_33, [
            _createElementVNode("div", _hoisted_34, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('provider.summaryProviders')), 1 /* TEXT */),
              _createElementVNode("strong", null, _toDisplayString(_ctx.providers.length), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.summaryProvidersCopy')), 1 /* TEXT */)
            ]),
            _createElementVNode("div", _hoisted_35, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('provider.summaryKeys')), 1 /* TEXT */),
              _createElementVNode("strong", null, _toDisplayString(_ctx.keys.length), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.summaryKeysCopy')), 1 /* TEXT */)
            ]),
            _createElementVNode("div", _hoisted_36, [
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
                  ? (_openBlock(), _createElementBlock("div", _hoisted_37, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                  : _createCommentVNode("v-if", true),
                _withDirectives(_createElementVNode("input", {
                  "onUpdate:modelValue": _cache[9] || (_cache[9] = $event => ((_ctx.providerForm.id) = $event)),
                  disabled: !!_ctx.providerForm.editingId,
                  placeholder: _ctx.t('placeholder.providerId')
                }, null, 8 /* PROPS */, _hoisted_38), [
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
                }, null, 8 /* PROPS */, _hoisted_39), [
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
                    }, _toDisplayString(type.label), 9 /* TEXT, PROPS */, _hoisted_40))
                  }), 128 /* KEYED_FRAGMENT */))
                ], 512 /* NEED_PATCH */), [
                  [_vModelSelect, _ctx.providerForm.type]
                ]),
                _withDirectives(_createElementVNode("input", {
                  "onUpdate:modelValue": _cache[12] || (_cache[12] = $event => ((_ctx.providerForm.baseUrl) = $event)),
                  placeholder: _ctx.t('placeholder.baseUrl')
                }, null, 8 /* PROPS */, _hoisted_41), [
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
                }, null, 8 /* PROPS */, _hoisted_42), [
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
                  placeholder: _ctx.t('placeholder.priority')
                }, null, 8 /* PROPS */, _hoisted_43), [
                  [
                    _vModelText,
                    _ctx.providerForm.priority,
                    void 0,
                    { number: true }
                  ]
                ]),
                _createElementVNode("div", _hoisted_44, [
                  _withDirectives(_createElementVNode("select", {
                    "onUpdate:modelValue": _cache[15] || (_cache[15] = $event => ((_ctx.providerForm.defaultModel) = $event))
                  }, [
                    _createElementVNode("option", _hoisted_45, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                    (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.providerForm.id || _ctx.providerForm.editingId), (model) => {
                      return (_openBlock(), _createElementBlock("option", {
                        key: model,
                        value: model
                      }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_46))
                    }), 128 /* KEYED_FRAGMENT */))
                  ], 512 /* NEED_PATCH */), [
                    [_vModelSelect, _ctx.providerForm.defaultModel]
                  ]),
                  _createElementVNode("small", null, _toDisplayString(_ctx.t('model.providerHint')), 1 /* TEXT */)
                ]),
                _createElementVNode("div", _hoisted_47, [
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
                    }, null, 8 /* PROPS */, _hoisted_48)), [
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
                    }, null, 8 /* PROPS */, _hoisted_49)), [
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
                      placeholder: _ctx.t('placeholder.keyPriority')
                    }, null, 8 /* PROPS */, _hoisted_50)), [
                      [
                        _vModelText,
                        _ctx.providerForm.firstKeyPriority,
                        void 0,
                        { number: true }
                      ]
                    ])
                  : _createCommentVNode("v-if", true),
                _createElementVNode("details", _hoisted_51, [
                  _createElementVNode("summary", null, _toDisplayString(_ctx.t('provider.modelMapTitle')), 1 /* TEXT */),
                  _createElementVNode("p", null, _toDisplayString(_ctx.t('provider.modelMapCopy')), 1 /* TEXT */),
                  _withDirectives(_createElementVNode("textarea", {
                    "onUpdate:modelValue": _cache[19] || (_cache[19] = $event => ((_ctx.providerForm.modelMap) = $event)),
                    spellcheck: "false",
                    placeholder: _ctx.t('placeholder.modelMap')
                  }, null, 8 /* PROPS */, _hoisted_52), [
                    [_vModelText, _ctx.providerForm.modelMap]
                  ]),
                  _createElementVNode("div", _hoisted_53, [
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
                _createElementVNode("div", _hoisted_54, [
                  _createElementVNode("button", {
                    class: "ghost",
                    type: "button",
                    "data-testid": "cancel-provider",
                    onClick: _cache[22] || (_cache[22] = (...args) => (_ctx.cancelProviderForm && _ctx.cancelProviderForm(...args)))
                  }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                  _createElementVNode("button", _hoisted_55, _toDisplayString(_ctx.t(_ctx.providerForm.editingId ? 'action.updateProvider' : 'action.saveProvider')), 1 /* TEXT */)
                ])
              ], 32 /* NEED_HYDRATION */))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_56, [
            (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (p) => {
              return (_openBlock(), _createElementBlock("article", {
                key: p.id,
                class: "provider-card"
              }, [
                _createElementVNode("div", _hoisted_57, [
                  _createElementVNode("div", null, [
                    _createElementVNode("div", _hoisted_58, [
                      _createElementVNode("code", null, _toDisplayString(p.id), 1 /* TEXT */),
                      _createElementVNode("span", {
                        class: _normalizeClass(['pill', p.enabled ? 'ok' : 'bad'])
                      }, _toDisplayString(p.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */),
                      _createElementVNode("span", _hoisted_59, _toDisplayString(_ctx.providerTypeLabel(p.type)), 1 /* TEXT */)
                    ]),
                    _createElementVNode("h3", null, _toDisplayString(p.name), 1 /* TEXT */),
                    _createElementVNode("small", _hoisted_60, _toDisplayString(p.baseUrl), 1 /* TEXT */)
                  ]),
                  _createElementVNode("div", _hoisted_61, [
                    _createElementVNode("button", {
                      type: "button",
                      onClick: $event => (_ctx.openKeyForm(p))
                    }, _toDisplayString(_ctx.t('action.addKey')), 9 /* TEXT, PROPS */, _hoisted_62),
                    _createElementVNode("button", {
                      class: "ghost",
                      type: "button",
                      onClick: $event => (_ctx.editProvider(p))
                    }, _toDisplayString(_ctx.t('action.editProvider')), 9 /* TEXT, PROPS */, _hoisted_63),
                    _createElementVNode("button", {
                      class: "ghost",
                      type: "button",
                      onClick: $event => (_ctx.toggleProvider(p))
                    }, _toDisplayString(_ctx.t('action.toggle')), 9 /* TEXT, PROPS */, _hoisted_64),
                    _createElementVNode("button", {
                      class: "danger",
                      type: "button",
                      onClick: $event => (_ctx.deleteProvider(p))
                    }, _toDisplayString(_ctx.t('action.delete')), 9 /* TEXT, PROPS */, _hoisted_65)
                  ])
                ]),
                (_ctx.providerFormOpen && _ctx.providerForm.editingId === p.id)
                  ? (_openBlock(), _createElementBlock("form", {
                      key: 0,
                      class: "form-grid inline-edit-form provider-inline-form",
                      "data-testid": "provider-form",
                      onSubmit: _cache[35] || (_cache[35] = _withModifiers((...args) => (_ctx.saveProvider && _ctx.saveProvider(...args)), ["prevent"]))
                    }, [
                      _createElementVNode("div", _hoisted_66, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.t('action.editProvider')), 1 /* TEXT */),
                        _createElementVNode("small", null, _toDisplayString(p.name) + " · " + _toDisplayString(p.baseUrl), 1 /* TEXT */)
                      ]),
                      (_ctx.formError)
                        ? (_openBlock(), _createElementBlock("div", _hoisted_67, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                        : _createCommentVNode("v-if", true),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[24] || (_cache[24] = $event => ((_ctx.providerForm.id) = $event)),
                        disabled: "",
                        placeholder: _ctx.t('placeholder.providerId')
                      }, null, 8 /* PROPS */, _hoisted_68), [
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
                      }, null, 8 /* PROPS */, _hoisted_69), [
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
                          }, _toDisplayString(type.label), 9 /* TEXT, PROPS */, _hoisted_70))
                        }), 128 /* KEYED_FRAGMENT */))
                      ], 512 /* NEED_PATCH */), [
                        [_vModelSelect, _ctx.providerForm.type]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[27] || (_cache[27] = $event => ((_ctx.providerForm.baseUrl) = $event)),
                        placeholder: _ctx.t('placeholder.baseUrl')
                      }, null, 8 /* PROPS */, _hoisted_71), [
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
                      }, null, 8 /* PROPS */, _hoisted_72), [
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
                        placeholder: _ctx.t('placeholder.priority')
                      }, null, 8 /* PROPS */, _hoisted_73), [
                        [
                          _vModelText,
                          _ctx.providerForm.priority,
                          void 0,
                          { number: true }
                        ]
                      ]),
                      _createElementVNode("div", _hoisted_74, [
                        _withDirectives(_createElementVNode("select", {
                          "onUpdate:modelValue": _cache[30] || (_cache[30] = $event => ((_ctx.providerForm.defaultModel) = $event))
                        }, [
                          _createElementVNode("option", _hoisted_75, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                          (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.providerForm.id || _ctx.providerForm.editingId), (model) => {
                            return (_openBlock(), _createElementBlock("option", {
                              key: model,
                              value: model
                            }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_76))
                          }), 128 /* KEYED_FRAGMENT */))
                        ], 512 /* NEED_PATCH */), [
                          [_vModelSelect, _ctx.providerForm.defaultModel]
                        ]),
                        _createElementVNode("small", null, _toDisplayString(_ctx.t('model.providerHint')), 1 /* TEXT */)
                      ]),
                      _createElementVNode("details", _hoisted_77, [
                        _createElementVNode("summary", null, _toDisplayString(_ctx.t('provider.modelMapTitle')), 1 /* TEXT */),
                        _createElementVNode("p", null, _toDisplayString(_ctx.t('provider.modelMapCopy')), 1 /* TEXT */),
                        _withDirectives(_createElementVNode("textarea", {
                          "onUpdate:modelValue": _cache[31] || (_cache[31] = $event => ((_ctx.providerForm.modelMap) = $event)),
                          spellcheck: "false",
                          placeholder: _ctx.t('placeholder.modelMap')
                        }, null, 8 /* PROPS */, _hoisted_78), [
                          [_vModelText, _ctx.providerForm.modelMap]
                        ]),
                        _createElementVNode("div", _hoisted_79, [
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
                      _createElementVNode("div", _hoisted_80, [
                        _createElementVNode("button", {
                          class: "ghost",
                          type: "button",
                          "data-testid": "cancel-provider",
                          onClick: _cache[34] || (_cache[34] = (...args) => (_ctx.cancelProviderForm && _ctx.cancelProviderForm(...args)))
                        }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                        _createElementVNode("button", _hoisted_81, _toDisplayString(_ctx.t('action.updateProvider')), 1 /* TEXT */)
                      ])
                    ], 32 /* NEED_HYDRATION */))
                  : _createCommentVNode("v-if", true),
                _createElementVNode("div", _hoisted_82, [
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
                (_ctx.keyFormOpen && !_ctx.keyForm.editingId && _ctx.keyForm.providerId === p.id)
                  ? (_openBlock(), _createElementBlock("form", {
                      key: 1,
                      class: "form-grid inline-edit-form key-inline-form",
                      "data-testid": "key-form",
                      onSubmit: _cache[43] || (_cache[43] = _withModifiers((...args) => (_ctx.saveKey && _ctx.saveKey(...args)), ["prevent"]))
                    }, [
                      _createElementVNode("div", _hoisted_83, [
                        _createElementVNode("strong", null, _toDisplayString(_ctx.t('action.addKey')), 1 /* TEXT */),
                        _createElementVNode("small", null, _toDisplayString(p.name) + " · " + _toDisplayString(p.baseUrl), 1 /* TEXT */)
                      ]),
                      (_ctx.formError)
                        ? (_openBlock(), _createElementBlock("div", _hoisted_84, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                        : _createCommentVNode("v-if", true),
                      _withDirectives(_createElementVNode("select", {
                        "onUpdate:modelValue": _cache[36] || (_cache[36] = $event => ((_ctx.keyForm.providerId) = $event)),
                        disabled: ""
                      }, [
                        (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (provider) => {
                          return (_openBlock(), _createElementBlock("option", {
                            key: provider.id,
                            value: provider.id
                          }, _toDisplayString(provider.name) + " (" + _toDisplayString(provider.id) + ")", 9 /* TEXT, PROPS */, _hoisted_85))
                        }), 128 /* KEYED_FRAGMENT */))
                      ], 512 /* NEED_PATCH */), [
                        [_vModelSelect, _ctx.keyForm.providerId]
                      ]),
                      _withDirectives(_createElementVNode("input", {
                        "onUpdate:modelValue": _cache[37] || (_cache[37] = $event => ((_ctx.keyForm.name) = $event)),
                        placeholder: _ctx.t('placeholder.keyName')
                      }, null, 8 /* PROPS */, _hoisted_86), [
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
                      }, null, 8 /* PROPS */, _hoisted_87), [
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
                        placeholder: _ctx.t('placeholder.priority')
                      }, null, 8 /* PROPS */, _hoisted_88), [
                        [
                          _vModelText,
                          _ctx.keyForm.priority,
                          void 0,
                          { number: true }
                        ]
                      ]),
                      _createElementVNode("div", _hoisted_89, [
                        _withDirectives(_createElementVNode("select", {
                          "onUpdate:modelValue": _cache[40] || (_cache[40] = $event => ((_ctx.keyForm.defaultModel) = $event))
                        }, [
                          _createElementVNode("option", _hoisted_90, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                          (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.keyForm.providerId), (model) => {
                            return (_openBlock(), _createElementBlock("option", {
                              key: model,
                              value: model
                            }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_91))
                          }), 128 /* KEYED_FRAGMENT */))
                        ], 512 /* NEED_PATCH */), [
                          [_vModelSelect, _ctx.keyForm.defaultModel]
                        ]),
                        _createElementVNode("small", null, _toDisplayString(_ctx.t('model.keyHint')), 1 /* TEXT */)
                      ]),
                      _createElementVNode("label", _hoisted_92, [
                        _withDirectives(_createElementVNode("input", {
                          "onUpdate:modelValue": _cache[41] || (_cache[41] = $event => ((_ctx.keyForm.enabled) = $event)),
                          type: "checkbox"
                        }, null, 512 /* NEED_PATCH */), [
                          [_vModelCheckbox, _ctx.keyForm.enabled]
                        ]),
                        _createTextVNode(" " + _toDisplayString(_ctx.t('status.enabled')), 1 /* TEXT */)
                      ]),
                      _createElementVNode("div", _hoisted_93, [
                        _createElementVNode("button", {
                          class: "ghost",
                          type: "button",
                          "data-testid": "cancel-key",
                          onClick: _cache[42] || (_cache[42] = (...args) => (_ctx.cancelKeyForm && _ctx.cancelKeyForm(...args)))
                        }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                        _createElementVNode("button", _hoisted_94, _toDisplayString(_ctx.t('action.saveKey')), 1 /* TEXT */)
                      ])
                    ], 32 /* NEED_HYDRATION */))
                  : _createCommentVNode("v-if", true),
                (_ctx.providerKeys(p.id).length)
                  ? (_openBlock(), _createElementBlock("div", _hoisted_95, [
                      (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerKeys(p.id), (k) => {
                        return (_openBlock(), _createElementBlock("div", {
                          key: k.id,
                          class: "key-card"
                        }, [
                          _createElementVNode("div", _hoisted_96, [
                            _createElementVNode("div", null, [
                              _createElementVNode("div", _hoisted_97, [
                                _createElementVNode("strong", _hoisted_98, _toDisplayString(k.name || k.id), 1 /* TEXT */),
                                (k.manualPreferred)
                                  ? (_openBlock(), _createElementBlock("span", _hoisted_99, _toDisplayString(_ctx.t('status.preferred')), 1 /* TEXT */))
                                  : _createCommentVNode("v-if", true),
                                _createElementVNode("span", {
                                  class: _normalizeClass(['pill', k.enabled ? 'ok' : 'bad'])
                                }, _toDisplayString(k.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */)
                              ]),
                              _createElementVNode("small", null, _toDisplayString(_ctx.t('provider.keyHealth', { ok: k.successCount || 0, fail: k.failureCount || 0 })), 1 /* TEXT */)
                            ]),
                            _createElementVNode("code", _hoisted_100, _toDisplayString(k.keyHint), 1 /* TEXT */)
                          ]),
                          (_ctx.keyFormOpen && _ctx.keyForm.editingId === k.id)
                            ? (_openBlock(), _createElementBlock("form", {
                                key: 0,
                                class: "form-grid inline-edit-form key-inline-form",
                                "data-testid": "key-form",
                                onSubmit: _cache[51] || (_cache[51] = _withModifiers((...args) => (_ctx.saveKey && _ctx.saveKey(...args)), ["prevent"]))
                              }, [
                                _createElementVNode("div", _hoisted_101, [
                                  _createElementVNode("strong", null, _toDisplayString(_ctx.t('action.editKey')), 1 /* TEXT */),
                                  _createElementVNode("small", null, _toDisplayString(p.name) + " · " + _toDisplayString(k.keyHint), 1 /* TEXT */)
                                ]),
                                (_ctx.formError)
                                  ? (_openBlock(), _createElementBlock("div", _hoisted_102, _toDisplayString(_ctx.formError), 1 /* TEXT */))
                                  : _createCommentVNode("v-if", true),
                                _withDirectives(_createElementVNode("select", {
                                  "onUpdate:modelValue": _cache[44] || (_cache[44] = $event => ((_ctx.keyForm.providerId) = $event)),
                                  disabled: ""
                                }, [
                                  (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (provider) => {
                                    return (_openBlock(), _createElementBlock("option", {
                                      key: provider.id,
                                      value: provider.id
                                    }, _toDisplayString(provider.name) + " (" + _toDisplayString(provider.id) + ")", 9 /* TEXT, PROPS */, _hoisted_103))
                                  }), 128 /* KEYED_FRAGMENT */))
                                ], 512 /* NEED_PATCH */), [
                                  [_vModelSelect, _ctx.keyForm.providerId]
                                ]),
                                _withDirectives(_createElementVNode("input", {
                                  "onUpdate:modelValue": _cache[45] || (_cache[45] = $event => ((_ctx.keyForm.name) = $event)),
                                  placeholder: _ctx.t('placeholder.keyName')
                                }, null, 8 /* PROPS */, _hoisted_104), [
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
                                }, null, 8 /* PROPS */, _hoisted_105), [
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
                                  placeholder: _ctx.t('placeholder.priority')
                                }, null, 8 /* PROPS */, _hoisted_106), [
                                  [
                                    _vModelText,
                                    _ctx.keyForm.priority,
                                    void 0,
                                    { number: true }
                                  ]
                                ]),
                                _createElementVNode("div", _hoisted_107, [
                                  _withDirectives(_createElementVNode("select", {
                                    "onUpdate:modelValue": _cache[48] || (_cache[48] = $event => ((_ctx.keyForm.defaultModel) = $event))
                                  }, [
                                    _createElementVNode("option", _hoisted_108, _toDisplayString(_ctx.t('model.useRequestDefault')), 1 /* TEXT */),
                                    (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providerModelOptions(_ctx.keyForm.providerId), (model) => {
                                      return (_openBlock(), _createElementBlock("option", {
                                        key: model,
                                        value: model
                                      }, _toDisplayString(model), 9 /* TEXT, PROPS */, _hoisted_109))
                                    }), 128 /* KEYED_FRAGMENT */))
                                  ], 512 /* NEED_PATCH */), [
                                    [_vModelSelect, _ctx.keyForm.defaultModel]
                                  ]),
                                  _createElementVNode("small", null, _toDisplayString(_ctx.t('model.keyHint')), 1 /* TEXT */)
                                ]),
                                _createElementVNode("label", _hoisted_110, [
                                  _withDirectives(_createElementVNode("input", {
                                    "onUpdate:modelValue": _cache[49] || (_cache[49] = $event => ((_ctx.keyForm.enabled) = $event)),
                                    type: "checkbox"
                                  }, null, 512 /* NEED_PATCH */), [
                                    [_vModelCheckbox, _ctx.keyForm.enabled]
                                  ]),
                                  _createTextVNode(" " + _toDisplayString(_ctx.t('status.enabled')), 1 /* TEXT */)
                                ]),
                                _createElementVNode("div", _hoisted_111, [
                                  _createElementVNode("button", {
                                    class: "ghost",
                                    type: "button",
                                    "data-testid": "cancel-key",
                                    onClick: _cache[50] || (_cache[50] = (...args) => (_ctx.cancelKeyForm && _ctx.cancelKeyForm(...args)))
                                  }, _toDisplayString(_ctx.t('action.cancel')), 1 /* TEXT */),
                                  _createElementVNode("button", _hoisted_112, _toDisplayString(_ctx.t('action.updateKey')), 1 /* TEXT */)
                                ])
                              ], 32 /* NEED_HYDRATION */))
                            : _createCommentVNode("v-if", true),
                          _createElementVNode("div", _hoisted_113, [
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.testKey(k)),
                              disabled: _ctx.testingKeyId === k.id
                            }, _toDisplayString(_ctx.testingKeyId === k.id ? _ctx.t('action.testing') : _ctx.t('action.test')), 9 /* TEXT, PROPS */, _hoisted_114),
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
                              ? (_openBlock(), _createElementBlock("small", _hoisted_115, _toDisplayString(_ctx.testErrorText(_ctx.keyTestResults[k.id])), 1 /* TEXT */))
                              : _createCommentVNode("v-if", true)
                          ]),
                          _createElementVNode("div", _hoisted_116, [
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.editKey(k))
                            }, _toDisplayString(_ctx.t('action.editKey')), 9 /* TEXT, PROPS */, _hoisted_117),
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.preferKey(k))
                            }, _toDisplayString(_ctx.t('action.prefer')), 9 /* TEXT, PROPS */, _hoisted_118),
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.resetKey(k))
                            }, _toDisplayString(_ctx.t('action.reset')), 9 /* TEXT, PROPS */, _hoisted_119),
                            _createElementVNode("button", {
                              class: "ghost",
                              type: "button",
                              onClick: $event => (_ctx.toggleKey(k))
                            }, _toDisplayString(_ctx.t('action.toggle')), 9 /* TEXT, PROPS */, _hoisted_120),
                            _createElementVNode("button", {
                              class: "danger",
                              type: "button",
                              onClick: $event => (_ctx.deleteKey(k))
                            }, _toDisplayString(_ctx.t('action.delete')), 9 /* TEXT, PROPS */, _hoisted_121)
                          ])
                        ]))
                      }), 128 /* KEYED_FRAGMENT */))
                    ]))
                  : (_openBlock(), _createElementBlock("div", _hoisted_122, [
                      _createElementVNode("strong", null, _toDisplayString(_ctx.t('empty.keys')), 1 /* TEXT */),
                      _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.keysHint')), 1 /* TEXT */),
                      _createElementVNode("button", {
                        type: "button",
                        onClick: $event => (_ctx.openKeyForm(p))
                      }, _toDisplayString(_ctx.t('action.addKey')), 9 /* TEXT, PROPS */, _hoisted_123)
                    ]))
              ]))
            }), 128 /* KEYED_FRAGMENT */)),
            (!_ctx.providers.length)
              ? (_openBlock(), _createElementBlock("div", _hoisted_124, [
                  _createElementVNode("strong", null, _toDisplayString(_ctx.t('empty.providers')), 1 /* TEXT */),
                  _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.providersHint')), 1 /* TEXT */),
                  _createElementVNode("button", {
                    type: "button",
                    onClick: _cache[52] || (_cache[52] = (...args) => (_ctx.openProviderForm && _ctx.openProviderForm(...args)))
                  }, _toDisplayString(_ctx.t('action.addProvider')), 1 /* TEXT */)
                ]))
              : _createCommentVNode("v-if", true)
          ]),
          _createElementVNode("div", _hoisted_125, [
            _createElementVNode("div", null, [
              _createElementVNode("h2", null, _toDisplayString(_ctx.t('provider.balanceTitle')), 1 /* TEXT */),
              _createElementVNode("small", null, _toDisplayString(_ctx.t('empty.balancesHint')), 1 /* TEXT */)
            ]),
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              onClick: _cache[53] || (_cache[53] = (...args) => (_ctx.refreshBalances && _ctx.refreshBalances(...args))),
              disabled: _ctx.balanceRefreshing
            }, _toDisplayString(_ctx.balanceRefreshing ? _ctx.t('action.refreshingBalance') : _ctx.t('action.refreshBalance')), 9 /* TEXT, PROPS */, _hoisted_126)
          ]),
          (_ctx.balanceResults.length)
            ? (_openBlock(), _createElementBlock("div", _hoisted_127, [
                (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.balanceResults, (result) => {
                  return (_openBlock(), _createElementBlock("div", {
                    key: result.providerId + ':' + result.keyId,
                    class: "compact-row"
                  }, [
                    _createElementVNode("div", null, [
                      _createElementVNode("strong", null, _toDisplayString(result.providerId) + " " + _toDisplayString(result.keyId || ''), 1 /* TEXT */),
                      _createElementVNode("small", null, _toDisplayString(result.error || result.status), 1 /* TEXT */)
                    ]),
                    _createElementVNode("span", {
                      class: _normalizeClass(['pill', _ctx.resultStatusClass(result.status)])
                    }, _toDisplayString(result.status), 3 /* TEXT, CLASS */)
                  ]))
                }), 128 /* KEYED_FRAGMENT */))
              ]))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_128, [
            _createElementVNode("table", null, [
              _createElementVNode("thead", null, [
                _createElementVNode("tr", null, [
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.provider')), 1 /* TEXT */),
                  _cache[92] || (_cache[92] = _createElementVNode("th", null, "Key", -1 /* CACHED */)),
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
                    _createElementVNode("td", null, _toDisplayString(b.providerId), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(b.keyId || '-'), 1 /* TEXT */),
                    _createElementVNode("td", {
                      class: _normalizeClass(_ctx.balanceValueClass(b))
                    }, _toDisplayString(_ctx.displayBalanceValue(b)), 3 /* TEXT, CLASS */),
                    _createElementVNode("td", null, [
                      _createElementVNode("span", {
                        class: _normalizeClass(['pill', _ctx.balanceStatusClass(b)])
                      }, _toDisplayString(_ctx.balanceStatusText(b.status)), 3 /* TEXT, CLASS */)
                    ]),
                    _createElementVNode("td", null, _toDisplayString(b.error || '-'), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(_ctx.formatSingaporeTime(b.refreshedAt || b.updatedAt)), 1 /* TEXT */)
                  ]))
                }), 128 /* KEYED_FRAGMENT */)),
                (!_ctx.balances.length)
                  ? (_openBlock(), _createElementBlock("tr", _hoisted_129, [
                      _createElementVNode("td", _hoisted_130, [
                        _createElementVNode("div", _hoisted_131, [
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
      _withDirectives(_createElementVNode("section", _hoisted_132, [
        _createElementVNode("article", _hoisted_133, [
          _createElementVNode("div", _hoisted_134, [
            _createElementVNode("strong", null, _toDisplayString(_ctx.t('keys.title')), 1 /* TEXT */),
            _createElementVNode("span", null, _toDisplayString(_ctx.t('keys.copy')), 1 /* TEXT */)
          ]),
          _createElementVNode("form", {
            class: "form-grid gateway-key-form",
            onSubmit: _cache[55] || (_cache[55] = _withModifiers((...args) => (_ctx.createGatewayKey && _ctx.createGatewayKey(...args)), ["prevent"]))
          }, [
            _withDirectives(_createElementVNode("input", {
              "onUpdate:modelValue": _cache[54] || (_cache[54] = $event => ((_ctx.gatewayKeyForm.name) = $event)),
              placeholder: _ctx.t('placeholder.gatewayKeyName')
            }, null, 8 /* PROPS */, _hoisted_135), [
              [
                _vModelText,
                _ctx.gatewayKeyForm.name,
                void 0,
                { trim: true }
              ]
            ]),
            _createElementVNode("div", _hoisted_136, [
              _createElementVNode("button", _hoisted_137, _toDisplayString(_ctx.t('action.addGatewayKey')), 1 /* TEXT */)
            ])
          ], 32 /* NEED_HYDRATION */),
          (_ctx.createdGatewayKey?.plaintext)
            ? (_openBlock(), _createElementBlock("div", _hoisted_138, [
                _createElementVNode("strong", null, _toDisplayString(_ctx.t(_ctx.createdGatewayKey.event === 'rotated' ? 'gateway.rotatedTitle' : 'gateway.createdTitle')), 1 /* TEXT */),
                _createElementVNode("span", null, _toDisplayString(_ctx.t(_ctx.createdGatewayKey.event === 'rotated' ? 'gateway.rotatedCopy' : 'gateway.createdCopy')), 1 /* TEXT */),
                _createElementVNode("code", _hoisted_139, _toDisplayString(_ctx.createdGatewayKey.plaintext), 1 /* TEXT */),
                _createElementVNode("button", {
                  type: "button",
                  class: _normalizeClass(_ctx.copyButtonClass('createdGatewayKey')),
                  onClick: _cache[56] || (_cache[56] = $event => (_ctx.copyText(_ctx.createdGatewayKey.plaintext, 'createdGatewayKey')))
                }, _toDisplayString(_ctx.copyButtonText('createdGatewayKey')), 3 /* TEXT, CLASS */)
              ]))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_140, [
            _createElementVNode("strong", null, _toDisplayString(_ctx.t('gateway.compatTitle')), 1 /* TEXT */),
            _createElementVNode("span", null, _toDisplayString(_ctx.t('gateway.compatCopy')), 1 /* TEXT */)
          ]),
          (!_ctx.gatewayKeys.length)
            ? (_openBlock(), _createElementBlock("div", _hoisted_141, [
                _createElementVNode("strong", null, _toDisplayString(_ctx.t('nav.keys')), 1 /* TEXT */),
                _createElementVNode("small", null, _toDisplayString(_ctx.t('keys.copy')), 1 /* TEXT */)
              ]))
            : (_openBlock(), _createElementBlock("div", _hoisted_142, [
                _createElementVNode("table", null, [
                  _createElementVNode("thead", null, [
                    _createElementVNode("tr", null, [
                      _cache[93] || (_cache[93] = _createElementVNode("th", null, "ID", -1 /* CACHED */)),
                      _createElementVNode("th", null, _toDisplayString(_ctx.t('table.name')), 1 /* TEXT */),
                      _cache[94] || (_cache[94] = _createElementVNode("th", null, "Key", -1 /* CACHED */)),
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
                          _createElementVNode("code", _hoisted_143, _toDisplayString(k.keyHint), 1 /* TEXT */)
                        ]),
                        _createElementVNode("td", null, _toDisplayString(k.requestCount), 1 /* TEXT */),
                        _createElementVNode("td", null, _toDisplayString(_ctx.formatSingaporeTime(k.lastUsedAt)), 1 /* TEXT */),
                        _createElementVNode("td", null, [
                          _createElementVNode("span", {
                            class: _normalizeClass(['pill', k.enabled ? 'ok' : 'bad'])
                          }, _toDisplayString(k.enabled ? _ctx.t('status.enabled') : _ctx.t('status.disabled')), 3 /* TEXT, CLASS */)
                        ]),
                        _createElementVNode("td", _hoisted_144, [
                          _createElementVNode("button", {
                            class: "ghost",
                            type: "button",
                            onClick: $event => (_ctx.toggleGatewayKey(k))
                          }, _toDisplayString(_ctx.t('action.toggle')), 9 /* TEXT, PROPS */, _hoisted_145),
                          _createElementVNode("button", {
                            class: "ghost",
                            type: "button",
                            onClick: $event => (_ctx.rotateGatewayKey(k))
                          }, _toDisplayString(_ctx.t('action.rotate')), 9 /* TEXT, PROPS */, _hoisted_146),
                          _createElementVNode("button", {
                            class: "danger",
                            type: "button",
                            onClick: $event => (_ctx.deleteGatewayKey(k))
                          }, _toDisplayString(_ctx.t('action.delete')), 9 /* TEXT, PROPS */, _hoisted_147)
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
      _withDirectives(_createElementVNode("section", _hoisted_148, [
        _createElementVNode("article", _hoisted_149, [
          _createElementVNode("form", {
            class: "form-grid routing-form",
            onSubmit: _cache[71] || (_cache[71] = _withModifiers((...args) => (_ctx.saveRouting && _ctx.saveRouting(...args)), ["prevent"]))
          }, [
            _createElementVNode("label", null, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.failureThreshold')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[57] || (_cache[57] = $event => ((_ctx.routing.failureThreshold) = $event)),
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
                "onUpdate:modelValue": _cache[58] || (_cache[58] = $event => ((_ctx.routing.cooldownSeconds) = $event)),
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
              _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.authCooldownSeconds')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[59] || (_cache[59] = $event => ((_ctx.routing.authCooldownSeconds) = $event)),
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
              _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.retryPerRequest')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[60] || (_cache[60] = $event => ((_ctx.routing.retryPerRequest) = $event)),
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
                "onUpdate:modelValue": _cache[61] || (_cache[61] = $event => ((_ctx.routing.timeoutSeconds) = $event)),
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
              _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.streamIdleTimeoutSeconds')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[62] || (_cache[62] = $event => ((_ctx.routing.streamIdleTimeoutSeconds) = $event)),
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
                "onUpdate:modelValue": _cache[63] || (_cache[63] = $event => ((_ctx.routing.streamWriteTimeoutSeconds) = $event)),
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
              _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.maxConcurrentRequests')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[64] || (_cache[64] = $event => ((_ctx.routing.maxConcurrentRequests) = $event)),
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
            _createElementVNode("label", null, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('routing.maxConcurrentPerProvider')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[65] || (_cache[65] = $event => ((_ctx.routing.maxConcurrentPerProvider) = $event)),
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
                "onUpdate:modelValue": _cache[66] || (_cache[66] = $event => ((_ctx.routing.maxConcurrentPerKey) = $event)),
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
                "onUpdate:modelValue": _cache[67] || (_cache[67] = $event => ((_ctx.routing.queueTimeoutMilliseconds) = $event)),
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
            ]),
            _createElementVNode("label", _hoisted_150, [
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[68] || (_cache[68] = $event => ((_ctx.routing.streamRetryBeforeFirstByte) = $event)),
                type: "checkbox"
              }, null, 512 /* NEED_PATCH */), [
                [_vModelCheckbox, _ctx.routing.streamRetryBeforeFirstByte]
              ]),
              _createTextVNode(" " + _toDisplayString(_ctx.t('routing.streamRetryBeforeFirstByte')), 1 /* TEXT */)
            ]),
            _createElementVNode("label", _hoisted_151, [
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[69] || (_cache[69] = $event => ((_ctx.routing.retryAmbiguousErrors) = $event)),
                type: "checkbox"
              }, null, 512 /* NEED_PATCH */), [
                [_vModelCheckbox, _ctx.routing.retryAmbiguousErrors]
              ]),
              _createTextVNode(" " + _toDisplayString(_ctx.t('routing.retryAmbiguousErrors')), 1 /* TEXT */)
            ]),
            _createElementVNode("label", _hoisted_152, [
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[70] || (_cache[70] = $event => ((_ctx.routing.allowInsecureUpstreams) = $event)),
                type: "checkbox"
              }, null, 512 /* NEED_PATCH */), [
                [_vModelCheckbox, _ctx.routing.allowInsecureUpstreams]
              ]),
              _createTextVNode(" " + _toDisplayString(_ctx.t('routing.allowInsecureUpstreams')), 1 /* TEXT */)
            ]),
            _createElementVNode("div", _hoisted_153, [
              _createElementVNode("button", _hoisted_154, _toDisplayString(_ctx.t('action.save')), 1 /* TEXT */)
            ])
          ], 32 /* NEED_HYDRATION */),
          (_ctx.routingMessage)
            ? (_openBlock(), _createElementBlock("div", _hoisted_155, _toDisplayString(_ctx.routingMessage), 1 /* TEXT */))
            : _createCommentVNode("v-if", true),
          _createElementVNode("div", _hoisted_156, [
            _createElementVNode("h2", null, _toDisplayString(_ctx.t('maintenance.title')), 1 /* TEXT */),
            _createElementVNode("div", _hoisted_157, [
              _createElementVNode("button", {
                class: "ghost",
                type: "button",
                onClick: _cache[72] || (_cache[72] = (...args) => (_ctx.checkIntegrity && _ctx.checkIntegrity(...args)))
              }, _toDisplayString(_ctx.t('maintenance.checkIntegrity')), 1 /* TEXT */),
              _createElementVNode("button", {
                class: "ghost",
                type: "button",
                onClick: _cache[73] || (_cache[73] = (...args) => (_ctx.downloadBackup && _ctx.downloadBackup(...args)))
              }, _toDisplayString(_ctx.t('maintenance.downloadDatabaseBackup')), 1 /* TEXT */)
            ])
          ]),
          _createElementVNode("form", {
            class: "portable-backup-form",
            onSubmit: _cache[76] || (_cache[76] = _withModifiers((...args) => (_ctx.downloadPortableBackup && _ctx.downloadPortableBackup(...args)), ["prevent"]))
          }, [
            _createElementVNode("label", null, [
              _createElementVNode("span", null, _toDisplayString(_ctx.t('maintenance.passphrase')), 1 /* TEXT */),
              _withDirectives(_createElementVNode("input", {
                "onUpdate:modelValue": _cache[74] || (_cache[74] = $event => ((_ctx.backupPassphrase) = $event)),
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
                "onUpdate:modelValue": _cache[75] || (_cache[75] = $event => ((_ctx.backupPassphraseConfirm) = $event)),
                type: "password",
                minlength: "12",
                autocomplete: "new-password",
                required: ""
              }, null, 512 /* NEED_PATCH */), [
                [_vModelText, _ctx.backupPassphraseConfirm]
              ])
            ]),
            _createElementVNode("button", _hoisted_158, _toDisplayString(_ctx.t('maintenance.downloadPortableBackup')), 1 /* TEXT */)
          ], 32 /* NEED_HYDRATION */),
          (_ctx.maintenanceMessage)
            ? (_openBlock(), _createElementBlock("div", _hoisted_159, _toDisplayString(_ctx.maintenanceMessage), 1 /* TEXT */))
            : _createCommentVNode("v-if", true)
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'routing']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_160, [
        _createElementVNode("article", _hoisted_161, [
          _createElementVNode("div", _hoisted_162, [
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              onClick: _cache[77] || (_cache[77] = $event => (_ctx.activeSection = 'providers'))
            }, _toDisplayString(_ctx.t('action.backProviders')), 1 /* TEXT */)
          ]),
          _createElementVNode("div", _hoisted_163, [
            _createElementVNode("strong", null, _toDisplayString(_ctx.t('setup.title')), 1 /* TEXT */),
            _createElementVNode("span", null, _toDisplayString(_ctx.t('setup.copy')), 1 /* TEXT */)
          ]),
          _createElementVNode("div", _hoisted_164, [
            (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.snippets, (value, name) => {
              return (_openBlock(), _createElementBlock("div", {
                key: name,
                class: "snippet"
              }, [
                _createElementVNode("div", _hoisted_165, [
                  _createElementVNode("h3", null, _toDisplayString(name), 1 /* TEXT */),
                  _createElementVNode("button", {
                    class: _normalizeClass(["ghost copy-chip", _ctx.copyButtonClass(`snippet:${name}`)]),
                    type: "button",
                    onClick: $event => (_ctx.copyText(typeof value === 'string' ? value : _ctx.pretty(value), `snippet:${name}`))
                  }, _toDisplayString(_ctx.copyButtonText(`snippet:${name}`)), 11 /* TEXT, CLASS, PROPS */, _hoisted_166)
                ]),
                _createElementVNode("pre", null, _toDisplayString(typeof value === 'string' ? value : _ctx.pretty(value)), 1 /* TEXT */)
              ]))
            }), 128 /* KEYED_FRAGMENT */))
          ])
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'setup']
      ]),
      _withDirectives(_createElementVNode("section", _hoisted_167, [
        _createElementVNode("article", _hoisted_168, [
          _createElementVNode("div", _hoisted_169, [
            _withDirectives(_createElementVNode("input", {
              "onUpdate:modelValue": _cache[78] || (_cache[78] = $event => ((_ctx.logFilters.q) = $event)),
              "aria-label": _ctx.t('logs.search'),
              placeholder: _ctx.t('logs.search'),
              onKeyup: _cache[79] || (_cache[79] = _withKeys($event => (_ctx.loadLogs(0)), ["enter"]))
            }, null, 40 /* PROPS, NEED_HYDRATION */, _hoisted_170), [
              [
                _vModelText,
                _ctx.logFilters.q,
                void 0,
                { trim: true }
              ]
            ]),
            _withDirectives(_createElementVNode("select", {
              "onUpdate:modelValue": _cache[80] || (_cache[80] = $event => ((_ctx.logFilters.status) = $event)),
              "aria-label": _ctx.t('table.status')
            }, [
              _createElementVNode("option", _hoisted_172, _toDisplayString(_ctx.t('logs.allStatus')), 1 /* TEXT */),
              _cache[95] || (_cache[95] = _createStaticVNode("<option value=\"200\">2xx</option><option value=\"400\">400</option><option value=\"401\">401</option><option value=\"429\">429</option><option value=\"499\">499</option><option value=\"500\">500</option><option value=\"502\">502</option><option value=\"503\">503</option>", 8))
            ], 8 /* PROPS */, _hoisted_171), [
              [_vModelSelect, _ctx.logFilters.status]
            ]),
            _withDirectives(_createElementVNode("select", {
              "onUpdate:modelValue": _cache[81] || (_cache[81] = $event => ((_ctx.logFilters.providerId) = $event)),
              "aria-label": _ctx.t('table.provider')
            }, [
              _createElementVNode("option", _hoisted_174, _toDisplayString(_ctx.t('logs.allProviders')), 1 /* TEXT */),
              (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.providers, (provider) => {
                return (_openBlock(), _createElementBlock("option", {
                  key: provider.id,
                  value: provider.id
                }, _toDisplayString(provider.name), 9 /* TEXT, PROPS */, _hoisted_175))
              }), 128 /* KEYED_FRAGMENT */))
            ], 8 /* PROPS */, _hoisted_173), [
              [_vModelSelect, _ctx.logFilters.providerId]
            ]),
            _withDirectives(_createElementVNode("input", {
              "onUpdate:modelValue": _cache[82] || (_cache[82] = $event => ((_ctx.logFilters.model) = $event)),
              "aria-label": _ctx.t('table.model'),
              placeholder: _ctx.t('table.model'),
              onKeyup: _cache[83] || (_cache[83] = _withKeys($event => (_ctx.loadLogs(0)), ["enter"]))
            }, null, 40 /* PROPS, NEED_HYDRATION */, _hoisted_176), [
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
              onClick: _cache[84] || (_cache[84] = $event => (_ctx.loadLogs(0)))
            }, _toDisplayString(_ctx.t('logs.apply')), 9 /* TEXT, PROPS */, _hoisted_177),
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              onClick: _cache[85] || (_cache[85] = (...args) => (_ctx.exportLogs && _ctx.exportLogs(...args)))
            }, _toDisplayString(_ctx.t('logs.export')), 1 /* TEXT */),
            _createElementVNode("button", {
              class: "danger",
              type: "button",
              onClick: _cache[86] || (_cache[86] = (...args) => (_ctx.clearLogs && _ctx.clearLogs(...args)))
            }, _toDisplayString(_ctx.t('logs.clear')), 1 /* TEXT */)
          ]),
          _createElementVNode("div", _hoisted_178, [
            _createElementVNode("table", null, [
              _createElementVNode("thead", null, [
                _createElementVNode("tr", null, [
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.time')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.protocol')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.model')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.provider')), 1 /* TEXT */),
                  _cache[96] || (_cache[96] = _createElementVNode("th", null, "Key", -1 /* CACHED */)),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.status')), 1 /* TEXT */),
                  _cache[97] || (_cache[97] = _createElementVNode("th", null, "Tokens", -1 /* CACHED */)),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.latency')), 1 /* TEXT */),
                  _createElementVNode("th", null, _toDisplayString(_ctx.t('table.error')), 1 /* TEXT */)
                ])
              ]),
              _createElementVNode("tbody", null, [
                (_openBlock(true), _createElementBlock(_Fragment, null, _renderList(_ctx.logs, (log) => {
                  return (_openBlock(), _createElementBlock("tr", {
                    key: log.id
                  }, [
                    _createElementVNode("td", null, _toDisplayString(_ctx.formatSingaporeTime(log.createdAt)), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.inboundProtocol), 1 /* TEXT */),
                    _createElementVNode("td", null, _toDisplayString(log.model || '-'), 1 /* TEXT */),
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
          _createElementVNode("div", _hoisted_179, [
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              disabled: _ctx.logPage.offset <= 0 || _ctx.logsLoading,
              onClick: _cache[87] || (_cache[87] = $event => (_ctx.loadLogs(_ctx.logPage.offset - _ctx.logPage.limit)))
            }, _toDisplayString(_ctx.t('logs.previous')), 9 /* TEXT, PROPS */, _hoisted_180),
            _createElementVNode("span", null, _toDisplayString(_ctx.logPage.total ? `${_ctx.logPage.offset + 1}-${Math.min(_ctx.logPage.offset + _ctx.logs.length, _ctx.logPage.total)} / ${_ctx.logPage.total}` : '0 / 0'), 1 /* TEXT */),
            _createElementVNode("button", {
              class: "ghost",
              type: "button",
              disabled: _ctx.logPage.offset + _ctx.logPage.limit >= _ctx.logPage.total || _ctx.logsLoading,
              onClick: _cache[88] || (_cache[88] = $event => (_ctx.loadLogs(_ctx.logPage.offset + _ctx.logPage.limit)))
            }, _toDisplayString(_ctx.t('logs.next')), 9 /* TEXT, PROPS */, _hoisted_181)
          ])
        ])
      ], 512 /* NEED_PATCH */), [
        [_vShow, _ctx.activeSection === 'logs']
      ])
    ])
  ], 64 /* STABLE_FRAGMENT */))
}
