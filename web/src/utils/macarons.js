/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

/* eslint-disable no-undef */
// eslint-disable-next-line no-extra-semi
const echarts = equire(['bar', 'legend', 'title', 'tooltip', 'pie']) // On-demand including
;(function(root, factory) {
  if (typeof define === 'function' && define.amd) {
    // AMD. Register as an anonymous module.
    define(['exports', 'echarts'], factory)
  } else if (typeof exports === 'object' && typeof exports.nodeName !== 'string') {
    // CommonJS
    factory(exports, echarts)
  } else {
    // Browser globals
    factory({}, echarts)
  }
})(this, function(exports, echarts) {
  var log = function(msg) {
    if (typeof console !== 'undefined') {
      console && console.error && console.error(msg)
    }
  }
  if (!echarts) {
    log('ECharts is not Loaded')
    return
  }

  var colorPalette = [
    '#2ec7c9',
    '#b6a2de',
    '#5ab1ef',
    '#ffb980',
    '#d87a80',
    '#8d98b3',
    '#e5cf0d',
    '#97b552',
    '#95706d',
    '#dc69aa',
    '#07a2a4',
    '#9a7fd1',
    '#588dd5',
    '#f5994e',
    '#c05050',
    '#59678c',
    '#c9ab00',
    '#7eb00a',
    '#6f5553',
    '#c14089'
  ]

  var theme = {
    color: colorPalette,

    title: {
      textStyle: {
        fontWeight: 'normal',
        color: '#008acd'
      }
    },

    visualMap: {
      itemWidth: 15,
      color: ['#5ab1ef', '#e0ffff']
    },

    toolbox: {
      iconStyle: {
        normal: {
          borderColor: colorPalette[0]
        }
      }
    },

    tooltip: {
      backgroundColor: 'rgba(50,50,50,0.5)',
      axisPointer: {
        type: 'line',
        lineStyle: {
          color: '#008acd'
        },
        crossStyle: {
          color: '#008acd'
        },
        shadowStyle: {
          color: 'rgba(200,200,200,0.2)'
        }
      }
    },

    dataZoom: {
      dataBackgroundColor: '#efefff',
      fillerColor: 'rgba(182,162,222,0.2)',
      handleColor: '#008acd'
    },

    grid: {
      borderColor: '#eee'
    },

    categoryAxis: {
      axisLine: {
        lineStyle: {
          color: '#008acd'
        }
      },
      splitLine: {
        lineStyle: {
          color: ['#eee']
        }
      }
    },

    valueAxis: {
      axisLine: {
        lineStyle: {
          color: '#008acd'
        }
      },
      splitArea: {
        show: true,
        areaStyle: {
          color: ['rgba(250,250,250,0.1)', 'rgba(200,200,200,0.1)']
        }
      },
      splitLine: {
        lineStyle: {
          color: ['#eee']
        }
      }
    },

    timeline: {
      lineStyle: {
        color: '#008acd'
      },
      controlStyle: {
        color: '#008acd',
        borderColor: '#008acd'
      },
      symbol: 'emptyCircle',
      symbolSize: 3
    },

    line: {
      smooth: true,
      symbol: 'emptyCircle',
      symbolSize: 3
    },

    candlestick: {
      itemStyle: {
        color: '#d87a80',
        color0: '#2ec7c9'
      },
      lineStyle: {
        width: 1,
        color: '#d87a80',
        color0: '#2ec7c9'
      },
      areaStyle: {
        color: '#2ec7c9',
        color0: '#b6a2de'
      }
    },

    scatter: {
      symbol: 'circle',
      symbolSize: 4
    },

    map: {
      itemStyle: {
        color: '#ddd'
      },
      areaStyle: {
        color: '#fe994e'
      },
      label: {
        color: '#d87a80'
      }
    },

    graph: {
      itemStyle: {
        color: '#d87a80'
      },
      linkStyle: {
        color: '#2ec7c9'
      }
    },

    gauge: {
      axisLine: {
        lineStyle: {
          color: [
            [0.2, '#2ec7c9'],
            [0.8, '#5ab1ef'],
            [1, '#d87a80']
          ],
          width: 10
        }
      },
      axisTick: {
        splitNumber: 10,
        length: 15,
        lineStyle: {
          color: 'auto'
        }
      },
      splitLine: {
        length: 22,
        lineStyle: {
          color: 'auto'
        }
      },
      pointer: {
        width: 5
      }
    }
  }

  echarts.registerTheme('macarons', theme)
})
