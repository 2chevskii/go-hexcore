import { defineConfig } from 'vitepress'

export default defineConfig({
  lang: 'en-US',
  title: 'go-hexcore',
  description: 'A Go client for managing HexCore servers, disks, and account resources.',
  base: '/go-hexcore/',
  cleanUrls: false,
  ignoreDeadLinks: false,
  srcExclude: ['examples/**'],
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'API reference', link: '/reference/client' },
      { text: 'Contribute', link: 'https://github.com/2chevskii/go-hexcore/blob/develop/.github/CONTRIBUTING.md' },
      { text: 'pkg.go.dev', link: 'https://pkg.go.dev/github.com/2chevskii/go-hexcore' },
    ],
    sidebar: [
      { text: 'Guides', items: [
        { text: 'Quick start', link: '/guide/getting-started' },
        { text: 'Authentication and configuration', link: '/guide/configuration' },
        { text: 'Contexts and deadlines', link: '/guide/contexts' },
        { text: 'Actions and resource readiness', link: '/guide/async' },
        { text: 'Errors and retries', link: '/guide/errors' },
      ] },
      { text: 'API reference', items: [
        { text: 'Client', link: '/reference/client' },
        { text: 'Account', link: '/reference/account' },
        { text: 'Catalogs', link: '/reference/catalogs' },
        { text: 'VDS', link: '/reference/vds' },
        { text: 'Disks', link: '/reference/disks' },
        { text: 'Actions', link: '/reference/actions' },
      ] },
      { text: 'Usage notes', items: [
        { text: 'Data types and units', link: '/data-conventions' },
        { text: 'API compatibility', link: '/compatibility' },
      ] },
    ],
    search: { provider: 'local' },
    socialLinks: [{ icon: 'github', link: 'https://github.com/2chevskii/go-hexcore' }],
    editLink: {
      pattern: 'https://github.com/2chevskii/go-hexcore/edit/develop/docs/:path',
    },
    outline: [2, 3],
    footer: { message: 'Released under the MIT License.' },
  },
})
