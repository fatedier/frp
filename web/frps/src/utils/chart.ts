import * as Humanize from 'humanize-plus'
import * as echarts from 'echarts/core'
import { PieChart, BarChart } from 'echarts/charts'
import { CanvasRenderer } from 'echarts/renderers'
import { LabelLayout } from 'echarts/features'

import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
} from 'echarts/components'

echarts.use([
  PieChart,
  BarChart,
  CanvasRenderer,
  LabelLayout,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
])

function DrawTrafficChart(
  elementId: string,
  trafficIn: number,
  trafficOut: number
) {
  const myChart = echarts.init(
    document.getElementById(elementId) as HTMLElement,
    'macarons'
  )
  myChart.showLoading()

  const option = {
    title: {
      text: 'Network Traffic',
      subtext: 'today',
      left: 'center',
    },
    tooltip: {
      trigger: 'item',
      formatter: function (v: any) {
        return Humanize.fileSize(v.data.value) + ' (' + v.percent + '%)'
      },
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      data: ['Traffic In', 'Traffic Out'],
    },
    series: [
      {
        type: 'pie',
        radius: '55%',
        center: ['50%', '60%'],
        data: [
          {
            value: trafficIn,
            name: 'Traffic In',
          },
          {
            value: trafficOut,
            name: 'Traffic Out',
          },
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  }
  myChart.setOption(option)
  myChart.hideLoading()
}

function DrawProxyChart(elementId: string, serverInfo: any) {
  const myChart = echarts.init(
    document.getElementById(elementId) as HTMLElement,
    'macarons'
  )
  myChart.showLoading()

  const option = {
    title: {
      text: 'Proxies',
      subtext: 'now',
      left: 'center',
    },
    tooltip: {
      trigger: 'item',
      formatter: function (v: any) {
        return String(v.data.value)
      },
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      data: <string[]>[],
    },
    series: [
      {
        type: 'pie',
        radius: '55%',
        center: ['50%', '60%'],
        data: <any[]>[],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  }

  if (
    serverInfo.proxy_type_count.tcp != null &&
    serverInfo.proxy_type_count.tcp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxy_type_count.tcp,
      name: 'TCP',
    })
    option.legend.data.push('TCP')
  }
  if (
    serverInfo.proxy_type_count.udp != null &&
    serverInfo.proxy_type_count.udp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxy_type_count.udp,
      name: 'UDP',
    })
    option.legend.data.push('UDP')
  }
  if (
    serverInfo.proxy_type_count.http != null &&
    serverInfo.proxy_type_count.http != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxy_type_count.http,
      name: 'HTTP',
    })
    option.legend.data.push('HTTP')
  }
  if (
    serverInfo.proxy_type_count.https != null &&
    serverInfo.proxy_type_count.https != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxy_type_count.https,
      name: 'HTTPS',
    })
    option.legend.data.push('HTTPS')
  }
  if (
    serverInfo.proxy_type_count.stcp != null &&
    serverInfo.proxy_type_count.stcp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxy_type_count.stcp,
      name: 'STCP',
    })
    option.legend.data.push('STCP')
  }
  if (
    serverInfo.proxy_type_count.sudp != null &&
    serverInfo.proxy_type_count.sudp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxy_type_count.sudp,
      name: 'SUDP',
    })
    option.legend.data.push('SUDP')
  }
  if (
    serverInfo.proxy_type_count.xtcp != null &&
    serverInfo.proxy_type_count.xtcp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxy_type_count.xtcp,
      name: 'XTCP',
    })
    option.legend.data.push('XTCP')
  }

  myChart.setOption(option)
  myChart.hideLoading()
}

// 7 days
function DrawProxyTrafficChart(
  elementId: string,
  trafficInArr: number[],
  trafficOutArr: number[]
) {
  const params = {
    width: '600px',
    height: '400px',
  }

  const myChart = echarts.init(
    document.getElementById(elementId) as HTMLElement,
    'macarons',
    params
  )
  myChart.showLoading()

  trafficInArr = trafficInArr.reverse()
  trafficOutArr = trafficOutArr.reverse()
  let now = new Date()
  now = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 6)
  const dates: Array<string> = []
  for (let i = 0; i < 7; i++) {
    dates.push(
      now.getFullYear() + '-' + (now.getMonth() + 1) + '-' + now.getDate()
    )
    now = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
  }

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow',
      },
      formatter: function (data: any) {
        let html = ''
        if (data.length > 0) {
          html += data[0].name + '<br/>'
        }
        for (const v of data) {
          const colorEl =
            '<span style="display:inline-block;margin-right:5px;' +
            'border-radius:10px;width:9px;height:9px;background-color:' +
            v.color +
            '"></span>'
          html +=
            colorEl + v.seriesName + ': ' + Humanize.fileSize(v.value) + '<br/>'
        }
        return html
      },
    },
    legend: {
      data: ['Traffic In', 'Traffic Out'],
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: [
      {
        type: 'category',
        data: dates,
      },
    ],
    yAxis: [
      {
        type: 'value',
        axisLabel: {
          formatter: function (value: number) {
            return Humanize.fileSize(value)
          },
        },
      },
    ],
    series: [
      {
        name: 'Traffic In',
        type: 'bar',
        data: trafficInArr,
      },
      {
        name: 'Traffic Out',
        type: 'bar',
        data: trafficOutArr,
      },
    ],
  }
  myChart.setOption(option)
  myChart.hideLoading()
}

export { DrawTrafficChart, DrawProxyChart, DrawProxyTrafficChart }
