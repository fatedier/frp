import { useBreakpoints } from '@vueuse/core'

const breakpoints = useBreakpoints({ mobile: 0, desktop: 768 })

export function useResponsive() {
  const isMobile = breakpoints.smaller('desktop') // < 768px
  return { isMobile }
}
