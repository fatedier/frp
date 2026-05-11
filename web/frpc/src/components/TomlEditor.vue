<template>
  <div ref="editorRef" class="toml-editor"></div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useDark } from '@vueuse/core'
import { basicSetup, EditorView } from 'codemirror'
import { EditorState, Extension, Compartment } from '@codemirror/state'
import { oneDark } from '@codemirror/theme-one-dark'
import { StreamLanguage } from '@codemirror/language'
import { toml } from '@codemirror/legacy-modes/mode/toml'
import { placeholder as placeholderExt } from '@codemirror/view'
import { keymap } from '@codemirror/view'
import { defaultKeymap, indentWithTab } from '@codemirror/commands'

const props = defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorRef = ref<HTMLDivElement>()
const editorView = ref<EditorView>()
const isDark = useDark()
const themeCompartment = new Compartment()

const content = computed(() => props.modelValue)

const lightTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: 'var(--color-bg-tertiary)',
      color: 'var(--color-text-primary)',
    },
    '.cm-content': {
      caretColor: 'var(--color-text-primary)',
    },
    '&.cm-focused .cm-cursor': {
      borderLeftColor: 'var(--color-text-primary)',
    },
    '&.cm-focused .cm-selectionBackground, ::selection': {
      backgroundColor: 'var(--color-primary-light)',
    },
    '.cm-gutters': {
      backgroundColor: 'var(--color-bg-muted)',
      color: 'var(--color-text-muted)',
      borderRight: '1px solid var(--color-border-light)',
    },
    '.cm-activeLineGutter': {
      backgroundColor: 'var(--color-bg-hover)',
    },
    '.cm-activeLine': {
      backgroundColor: 'var(--color-bg-hover)',
    },
    '.cm-lineNumbers': {
      color: 'var(--color-text-muted)',
    },
  },
  { dark: false }
)

const darkThemeOverride = EditorView.theme(
  {
    '&': {
      backgroundColor: 'var(--color-bg-tertiary)',
    },
    '.cm-gutters': {
      backgroundColor: 'var(--color-bg-muted)',
      borderRight: '1px solid var(--color-border-light)',
    },
  },
  { dark: true }
)

const createThemeExtensions = (): Extension[] => {
  return isDark.value ? [oneDark, darkThemeOverride] : [lightTheme]
}

const createExtensions = (): Extension[] => {
  const exts: Extension[] = [
    basicSetup,
    keymap.of([...defaultKeymap, indentWithTab]),
    StreamLanguage.define(toml),
    EditorView.updateListener.of((v) => {
      if (v.docChanged) {
        emit('update:modelValue', v.state.doc.toString())
      }
    }),
    EditorView.lineWrapping,
    themeCompartment.of(createThemeExtensions()),
  ]

  if (props.placeholder) {
    exts.push(placeholderExt(props.placeholder))
  }

  return exts
}

const initEditor = () => {
  if (!editorRef.value) return

  const state = EditorState.create({
    doc: content.value,
    extensions: createExtensions(),
  })

  editorView.value = new EditorView({
    state,
    parent: editorRef.value,
  })
}

const destroyEditor = () => {
  editorView.value?.destroy()
  editorView.value = undefined
}

const reconfigureTheme = () => {
  if (!editorView.value) return
  editorView.value.dispatch({
    effects: themeCompartment.reconfigure(createThemeExtensions()),
  })
}

watch(
  () => content.value,
  (newValue) => {
    if (!editorView.value) {
      initEditor()
      return
    }
    if (newValue === editorView.value.state.doc.toString()) {
      return
    }
    editorView.value.dispatch({
      changes: {
        from: 0,
        to: editorView.value.state.doc.length,
        insert: newValue,
      },
      scrollIntoView: false,
    })
  },
  { immediate: true }
)

watch(isDark, () => {
  reconfigureTheme()
})

onMounted(() => {
  if (!editorView.value) {
    initEditor()
  }
})

onUnmounted(() => {
  destroyEditor()
})
</script>

<style scoped lang="scss">
.toml-editor {
  height: 100%;

  :deep(.cm-editor) {
    height: 100%;
    border-radius: $radius-md;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: $font-size-sm;
    line-height: 1.6;
    border: 1px solid var(--color-border-light);
    overflow: hidden;
  }

  :deep(.cm-editor.cm-focused) {
    border-color: var(--color-text-light);
    outline: none;
  }

  :deep(.cm-scroller) {
    border-radius: $radius-md;
  }
}
</style>
