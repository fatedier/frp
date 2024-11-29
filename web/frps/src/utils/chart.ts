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
    serverInfo.proxyTypeCount.tcp != null &&
    serverInfo.proxyTypeCount.tcp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxyTypeCount.tcp,
      name: 'TCP',
    })
    option.legend.data.push('TCP')
  }
  if (
    serverInfo.proxyTypeCount.udp != null &&
    serverInfo.proxyTypeCount.udp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxyTypeCount.udp,
      name: 'UDP',
    })
    option.legend.data.push('UDP')
  }
  if (
    serverInfo.proxyTypeCount.http != null &&
    serverInfo.proxyTypeCount.http != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxyTypeCount.http,
      name: 'HTTP',
    })
    option.legend.data.push('HTTP')
  }
  if (
    serverInfo.proxyTypeCount.https != null &&
    serverInfo.proxyTypeCount.https != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxyTypeCount.https,
      name: 'HTTPS',
    })
    option.legend.data.push('HTTPS')
  }
  if (
    serverInfo.proxyTypeCount.stcp != null &&
    serverInfo.proxyTypeCount.stcp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxyTypeCount.stcp,
      name: 'STCP',
    })
    option.legend.data.push('STCP')
  }
  if (
    serverInfo.proxyTypeCount.sudp != null &&
    serverInfo.proxyTypeCount.sudp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxyTypeCount.sudp,
      name: 'SUDP',
    })
    option.legend.data.push('SUDP')
  }
  if (
    serverInfo.proxyTypeCount.xtcp != null &&
    serverInfo.proxyTypeCount.xtcp != 0
  ) {
    option.series[0].data.push({
      value: serverInfo.proxyTypeCount.xtcp,
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
