import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    projects: [
      {
        extends: './frpc/vite.config.mts',
        root: './frpc',
        test: {
          name: 'frpc',
          environment: 'jsdom',
          css: true,
          server: {
            deps: { inline: ['element-plus', '@element-plus/icons-vue'] },
          },
          setupFiles: ['../test/setup.ts'],
          include: [
            '../test/**/*.test.ts',
            'test/**/*.test.ts',
            '../shared/**/*.test.ts',
          ],
        },
      },
      {
        extends: './frps/vite.config.mts',
        root: './frps',
        test: {
          name: 'frps',
          environment: 'jsdom',
          css: true,
          server: {
            deps: { inline: ['element-plus', '@element-plus/icons-vue'] },
          },
          setupFiles: ['../test/setup.ts'],
          include: ['test/**/*.test.ts'],
        },
      },
    ],
  },
})
