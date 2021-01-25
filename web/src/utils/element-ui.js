import lang from 'element-ui/lib/locale/lang/en'
import locale from 'element-ui/lib/locale'
import 'element-ui/lib/theme-chalk/index.css'
import Vue from 'vue'

locale.use(lang)

import {
  Button,
  Input,
  Message,
  Form,
  FormItem,
  Col,
  Row,
  Breadcrumb,
  BreadcrumbItem,
  Menu,
  MenuItem,
  Submenu,
  Scrollbar,
  Table,
  TableColumn,
  Tag,
  Popover,
  MessageBox
} from 'element-ui'

Vue.use(Button)
Vue.use(Input)
Vue.use(Form)
Vue.use(FormItem)
Vue.use(Col)
Vue.use(Row)
Vue.use(Breadcrumb)
Vue.use(BreadcrumbItem)
Vue.use(Menu)
Vue.use(MenuItem)
Vue.use(Submenu)
Vue.use(Scrollbar)
Vue.use(Table)
Vue.use(TableColumn)
Vue.use(Tag)
Vue.use(Popover)

Vue.prototype.$message = Message
Vue.prototype.$confirm = MessageBox.confirm
