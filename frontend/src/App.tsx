import { useMemo, useState } from 'react'
import { Activity, Box, ChevronRight, Copy, Folder, History, LogOut, Moon, Search, Settings2, Sun, Tag, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog'

type Data = Record<string, any> // The server sends the existing page data as JSON.
type Bootstrap = { page: string; data: Data; basePath: string; registryHost: string; version: string }
const app: Bootstrap = JSON.parse(document.getElementById('app-data')?.textContent || '{}')
const base = app.basePath || ''
const url = (path: string) => `${base}/${path.replace(/^\/+/, '')}`
const date = (value: string | undefined) => {
  if (!value || value.startsWith('0001-')) return '—'
  const parsed = new Date(value.includes('T') || value.endsWith('Z') ? value : value.replace(' ', 'T') + 'Z')
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString('zh-CN', { hour12: false })
}
const size = (bytes: number) => {
  if (!bytes) return '0 B'
  const unit = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), 4)
  return `${(bytes / 1024 ** unit).toFixed(unit ? 1 : 0)} ${['B', 'KB', 'MB', 'GB', 'TB'][unit]}`
}

function Shell({ children }: { children: React.ReactNode }) {
  const [dark, setDark] = useState(() => localStorage.getItem('theme') === 'dark')
  document.documentElement.classList.toggle('dark', dark)
  const links = [
    { label: '镜像仓库', path: '', icon: Folder, active: ['catalog.html', 'image_info.html'].includes(app.page) },
    ...(app.data.eventsAllowed ? [
      { label: '运行概览', path: '__statistics', icon: Activity, active: app.page === 'statistics.html' },
      { label: '清理记录', path: '__purge-log', icon: History, active: app.page === 'purge_log.html' },
      { label: '系统配置', path: '__options', icon: Settings2, active: app.page === 'options.html' },
    ] : []),
  ]
  if (app.page === 'login.html') return <>{children}</>
  return <div className="min-h-screen lg:flex">
    <aside className="sticky top-0 z-20 border-b border-border bg-card lg:fixed lg:inset-y-0 lg:w-58 lg:border-r lg:border-b-0">
      <div className="flex items-center gap-3 px-4 pt-4 lg:px-5 lg:pt-7">
        <a href={url('')} className="flex min-w-0 flex-1 items-center gap-3"><span className="grid size-10 place-items-center rounded-xl bg-primary text-white"><Box size={21}/></span><span><strong className="block text-base">镜像仓库</strong><span className="text-[10px] tracking-widest text-muted-foreground">REGISTRY CONSOLE</span></span></a>
        <Button variant="ghost" size="icon" className="lg:hidden" aria-label="切换明暗模式" onClick={() => { localStorage.setItem('theme', dark ? 'light' : 'dark'); setDark(!dark) }}>{dark ? <Sun/> : <Moon/>}</Button>
        {app.data.user && <Button asChild variant="ghost" size="icon" className="lg:hidden"><a href={url('logout')} aria-label="退出登录"><LogOut/></a></Button>}
      </div>
      <nav aria-label="主导航" className="flex gap-1 overflow-x-auto px-3 py-3 lg:flex-col lg:pt-10">{links.map(({label,path,icon:Icon,active}) => <a key={path} href={url(path)} aria-current={active ? 'page' : undefined} className={`flex shrink-0 items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium ${active ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-secondary hover:text-foreground'}`}><Icon size={17}/>{label}</a>)}</nav>
      <div className="hidden items-center gap-2 border-t border-border p-4 lg:absolute lg:inset-x-0 lg:bottom-0 lg:flex"><Button variant="ghost" size="icon" aria-label="切换明暗模式" onClick={() => { localStorage.setItem('theme', dark ? 'light' : 'dark'); setDark(!dark) }}>{dark ? <Sun/> : <Moon/>}</Button>{app.data.user && <Button asChild variant="ghost"><a href={url('logout')}><LogOut/>退出登录</a></Button>}</div>
    </aside>
    <main className="min-w-0 flex-1 px-4 py-7 md:px-8 lg:ml-58 lg:px-10 lg:py-10"><div className="mx-auto max-w-350 space-y-5">{children}</div></main>
  </div>
}

function PageTitle({ title, path, heading, search, onSearch, action }: { title: string; path?: string; heading?: string; search?: string; onSearch?: (v: string) => void; action?: React.ReactNode }) {
  const parts = path ? path.split('/') : []
  return <header className="space-y-5"><div><div className="mb-2 text-xs text-muted-foreground">镜像管理 / {title}</div><h1 className="break-all text-2xl font-bold tracking-tight">{heading || path || title}</h1></div><div className="flex flex-wrap items-center justify-between gap-3"><nav className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground" aria-label="面包屑"><a href={url('')} className="hover:text-primary">首页</a>{parts.map((part, i) => { const current = parts.slice(0, i + 1).join('/'); return <span className="flex items-center gap-2" key={current}><ChevronRight size={14}/><a className="hover:text-primary" href={url(current)}>{part}</a></span> })}</nav>{onSearch && <div className="relative w-full sm:w-80"><Search size={16} className="absolute left-3 top-3 text-muted-foreground"/><Input aria-label="搜索" className="pl-9" placeholder="搜索当前列表" value={search} onChange={e => onSearch(e.target.value)}/></div>}{action}</div></header>
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return <Card className="gap-0 overflow-hidden py-0 shadow-xs"><CardHeader className="app-content-inset border-b border-border py-4 !pb-4"><CardTitle className="text-[15px]">{title}</CardTitle></CardHeader><CardContent className="px-0">{children}</CardContent></Card>
}
function Empty({ children }: { children: React.ReactNode }) { return <div className="app-content-inset py-10 text-center text-sm text-muted-foreground">{children}</div> }
function DeleteButton({ repo, tag }: { repo: string; tag: string }) {
  const deleteURL = `${url('__delete-tag')}?${new URLSearchParams({repoPath: repo, tag})}`
  return <AlertDialog><AlertDialogTrigger asChild><Button variant="ghost" size="icon-sm" aria-label={`删除 ${tag}`} title="删除标签"><Trash2 className="text-muted-foreground"/></Button></AlertDialogTrigger><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>确认删除标签 {tag}？</AlertDialogTitle><AlertDialogDescription>删除镜像可能同时影响指向同一镜像的其他标签，请确认它们不再使用。</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction variant="destructive" onClick={() => { window.location.href = deleteURL }}>确认删除</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
}
function ListPager({ total, page, setPage }: { total: number; page: number; setPage: (v: number) => void }) {
  const pages = Math.max(1, Math.ceil(total / 25))
  return <div className="app-content-inset flex flex-wrap items-center justify-between gap-3 border-t border-border py-3 text-xs text-muted-foreground"><span>共 {total} 条</span><div className="flex items-center gap-2"><Button variant="outline" size="sm" disabled={page <= 0} onClick={() => setPage(page - 1)}>上一页</Button><span>{Math.min(page + 1, pages)} / {pages}</span><Button variant="outline" size="sm" disabled={page + 1 >= pages} onClick={() => setPage(page + 1)}>下一页</Button></div></div>
}
function Catalog({ data }: { data: Data }) {
  const [search, setSearch] = useState(''); const [repoPage, setRepoPage] = useState(0); const [tagPage, setTagPage] = useState(0)
  const path: string = data.repoPath || ''
  const filter = (items: string[]) => items.filter(x => x.toLowerCase().includes(search.toLowerCase()))
  const repos = filter(data.repos || []); const tags = filter(data.tags || []).sort((a, b) => b.localeCompare(a, undefined, { numeric: true }))
  const updateSearch = (value: string) => { setSearch(value); setRepoPage(0); setTagPage(0) }
  const notice: Record<string, string> = { deleted: '标签删除请求已完成。指向同一镜像的其他标签也可能受到影响。', delete_failed: '删除失败，请检查 Registry 删除权限和服务日志。', delete_denied: '当前账号没有删除标签的权限。' }
  return <><PageTitle title="镜像仓库" path={path} search={search} onSearch={updateSearch}/>{notice[data.notice] && <div role="status" className="rounded-lg border border-border bg-card app-content-inset py-4">{notice[data.notice]}</div>}
    {(repos.length > 0 || !data.isCatalogReady || !path || (data.repos || []).length > 0) && <Section title="仓库">{repos.length ? <><Table><TableHeader><TableRow><TableHead>仓库</TableHead><TableHead className="w-32">标签数</TableHead></TableRow></TableHeader><TableBody>{repos.slice(repoPage * 25, (repoPage + 1) * 25).map(repo => { const full = path ? `${path}/${repo}` : repo; return <TableRow key={full}><TableCell><a className="flex items-center gap-3 font-medium hover:text-primary" href={url(full)}><span className="rounded-md bg-primary/10 p-2 text-primary"><Folder size={16}/></span>{repo}</a></TableCell><TableCell>{data.tagCounts?.[full] ?? '—'}</TableCell></TableRow> })}</TableBody></Table><ListPager total={repos.length} page={repoPage} setPage={setRepoPage}/></> : <Empty>{!data.isCatalogReady ? '镜像仓库正在初始化…' : search ? '没有找到匹配的仓库' : '暂无仓库'}</Empty>}</Section>}
    {data.showTags && <Section title="标签">{tags.length ? <><Table><TableHeader><TableRow><TableHead>标签名称</TableHead><TableHead className="w-20 text-right">操作</TableHead></TableRow></TableHeader><TableBody>{tags.slice(tagPage * 25, (tagPage + 1) * 25).map(tag => <TableRow key={tag}><TableCell><a className="flex items-center gap-3 font-medium hover:text-primary" href={url(`${path}:${encodeURIComponent(tag)}`)}><span className="rounded-md bg-primary/10 p-2 text-primary"><Tag size={16}/></span>{tag}</a></TableCell><TableCell className="text-right">{data.deleteAllowed && <DeleteButton repo={path} tag={tag}/>}</TableCell></TableRow>)}</TableBody></Table><ListPager total={tags.length} page={tagPage} setPage={setTagPage}/></> : <Empty>{!data.areTagsReady ? '标签正在加载…' : search ? '没有找到匹配的标签' : '暂无标签'}</Empty>}</Section>}
    {!!data.eventsAllowed && !!data.events?.length && <Section title="近期活动"><Table><TableHeader><TableRow>{['操作','镜像','用户','时间'].map(x=><TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader><TableBody>{data.events.map((e: Data, i: number)=><TableRow key={i}><TableCell><Badge variant="secondary">{e.Action}</Badge></TableCell><TableCell>{e.Repository}:{e.Tag}</TableCell><TableCell>{e.User || '—'}</TableCell><TableCell className="whitespace-nowrap">{date(e.Created)}</TableCell></TableRow>)}</TableBody></Table></Section>}
  </>
}

function ImageDetails({ data }: { data: Data }) {
  const image = data.ii || {}; const [copy, setCopy] = useState(''); const reference = `${app.registryHost}/${data.repoPath}`
  const copyReference = async () => { try { if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(reference); else { const el = document.createElement('textarea'); el.value = reference; document.body.appendChild(el); el.select(); if (!document.execCommand('copy')) throw Error(); el.remove() } setCopy('已复制') } catch { setCopy('复制失败，请手动复制') } }
  const fields = [['镜像引用', <span key="reference" className="flex flex-wrap items-center gap-2"><code className="break-all">{reference}</code><Button variant="ghost" size="icon-sm" title="复制镜像地址" aria-label="复制镜像地址" onClick={copyReference}><Copy/></Button><span role="status" className="text-xs text-muted-foreground">{copy}</span></span>], ['当前标签', image.ImageRefTag], ...(image.IsImage ? [['镜像大小', size(image.ImageSize)], ['创建时间', date(image.Created)]] : []), ['平台', image.Platforms || '—']]
  return <><PageTitle title="标签详情" path={image.ImageRefRepo} heading={image.ImageRefTag} action={data.deleteAllowed ? <DeleteButton repo={image.ImageRefRepo} tag={image.ImageRefTag}/> : undefined}/><Section title="概要信息"><Table><TableBody>{fields.map(([label,value]) => <TableRow key={String(label)}><TableCell className="w-36 font-medium text-muted-foreground">{label}</TableCell><TableCell>{value}</TableCell></TableRow>)}</TableBody></Table></Section>{[['技术信息', {摘要:image.ImageRefDigest, 媒体类型:image.MediaType, 镜像ID:image.ConfigImageID}], ['镜像清单', image.Manifest], ...(image.IsImage ? [['镜像配置', image.ConfigFile]] : [])].map(([title,value]) => <Card key={String(title)} className="gap-0 py-0 shadow-xs"><details><summary className="app-content-inset cursor-pointer py-4 font-semibold">{String(title)}</summary><pre className="max-h-120 overflow-auto app-content-inset border-t border-border py-5 text-xs">{JSON.stringify(value, null, 2)}</pre></details></Card>)}</>
}

function Statistics({ data }: { data: Data }) {
  const metrics = [['仓库', data.isCatalogReady ? data.repoCount : '—'], ['标签', data.tagCount], ['事件记录', data.eventCount]]
  return <><PageTitle title="运行概览"/><div className="grid gap-5 sm:grid-cols-3">{metrics.map(([label,value])=><Card key={String(label)} className="app-content-inset gap-3 py-5 shadow-xs"><div className="text-sm text-muted-foreground">{label}</div><div className="text-3xl font-bold">{value}</div></Card>)}</div><Section title="后台任务"><Table><TableHeader><TableRow><TableHead>任务</TableHead><TableHead>上次运行</TableHead><TableHead>耗时</TableHead></TableRow></TableHeader><TableBody>{[['目录刷新',data.catalogJobInfo],['标签刷新',data.tagsJobInfo]].map(([name,job]: any)=><TableRow key={name}><TableCell>{name}</TableCell><TableCell>{date(job?.LastRun)}</TableCell><TableCell>{job?.Duration ? `${Math.round(job.Duration / 1000000)} ms` : '—'}</TableCell></TableRow>)}</TableBody></Table></Section><Section title="定时清理"><div className="app-content-inset grid gap-5 py-5 sm:grid-cols-3"><div><div className="text-xs text-muted-foreground">清理计划</div><div className="mt-2">{data.purgeCronExpr || '未启用'}</div></div><div><div className="text-xs text-muted-foreground">最近执行</div><div className="mt-2">{date(data.latestPurgeRun?.StartedAt)}</div></div><div><div className="text-xs text-muted-foreground">结果</div><div className="mt-2">{data.latestPurgeRun ? (data.latestPurgeRun.Success ? '成功' : '失败') : '暂无记录'}</div></div></div></Section>{!!data.topRepos?.length && <Section title="标签数量最多的仓库"><Table><TableHeader><TableRow><TableHead>仓库</TableHead><TableHead>标签数</TableHead></TableRow></TableHeader><TableBody>{data.topRepos.map((repo: Data)=><TableRow key={repo.Repo}><TableCell><a className="font-medium hover:text-primary" href={url(repo.Repo)}>{repo.Repo}</a></TableCell><TableCell>{repo.Count}</TableCell></TableRow>)}</TableBody></Table></Section>}</>
}
function PurgeLog({ data }: { data: Data }) {
  const [search,setSearch] = useState(''); const [page,setPage] = useState(0)
  const rows = useMemo(() => (data.purgeRuns || []).filter((r: Data) => JSON.stringify(r).toLowerCase().includes(search.toLowerCase())), [data.purgeRuns, search])
  return <><PageTitle title="清理记录" search={search} onSearch={v => {setSearch(v);setPage(0)}}/><div className="rounded-lg border border-border bg-card app-content-inset py-3 text-sm text-muted-foreground">候选镜像大小不等于实际回收空间，实际回收量取决于共享镜像层和 Registry 垃圾回收。</div><Section title="执行记录">{rows.length ? <><Table><TableHeader><TableRow>{['开始时间','状态','模式','仓库','候选','已删除','候选大小','明细','错误'].map(x=><TableHead key={x} className="whitespace-nowrap">{x}</TableHead>)}</TableRow></TableHeader><TableBody>{rows.slice(page*25,(page+1)*25).map((r: Data)=><TableRow key={r.ID}><TableCell className="whitespace-nowrap">{date(r.StartedAt)}</TableCell><TableCell><Badge variant={r.Success ? 'secondary' : 'destructive'}>{r.Success ? '成功' : '失败'}</Badge></TableCell><TableCell>{r.DryRun ? '模拟' : '执行'}</TableCell><TableCell>{r.RepoCount}</TableCell><TableCell>{r.CandidateTagCount}</TableCell><TableCell>{r.DeletedTagCount}</TableCell><TableCell className="whitespace-nowrap">{size(r.EstimatedFreedBytes)}</TableCell><TableCell>{r.Summary && <details><summary className="cursor-pointer text-primary">查看</summary><pre className="max-w-100 text-xs">{r.Summary}</pre></details>}</TableCell><TableCell>{r.ErrorMessage ? <details><summary className="cursor-pointer text-destructive">查看</summary><pre className="max-w-100 text-xs">{r.ErrorMessage}</pre></details> : '—'}</TableCell></TableRow>)}</TableBody></Table><ListPager total={rows.length} page={page} setPage={setPage}/></> : <Empty>{search ? '没有找到匹配的清理记录' : '暂无清理记录'}</Empty>}</Section></>
}
function Options({ data }: { data: Data }) { return <><PageTitle title="系统配置"/>{(data.sections || []).map((section: Data)=><Section key={section.Name} title={section.Name}><Table><TableBody>{section.Options.map(([key,value]: string[])=><TableRow key={key}><TableCell className="w-1/2 font-medium text-muted-foreground">{key}</TableCell><TableCell className="break-all">{value}</TableCell></TableRow>)}</TableBody></Table></Section>)}</> }
function Login({ data }: { data: Data }) { return <div className="flex min-h-screen items-center justify-center p-5"><Card className="w-full max-w-md p-8 shadow-sm"><div className="mb-2 flex items-center gap-3"><span className="grid size-11 place-items-center rounded-xl bg-primary text-white"><Box/></span><div><h1 className="text-xl font-bold">登录镜像仓库</h1><p className="text-sm text-muted-foreground">使用管理员账号登录</p></div></div>{data.error && <div role="alert" className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{data.error}</div>}<form method="POST" action={url('login')} className="space-y-4"><label className="block space-y-2"><span className="text-sm font-medium">用户名</span><Input name="username" autoComplete="username" required autoFocus/></label><label className="block space-y-2"><span className="text-sm font-medium">密码</span><Input name="password" type="password" autoComplete="current-password" required/></label><Button type="submit" className="w-full">登录</Button></form></Card></div> }
export default function App() {
  const data = app.data || {}
  let content: React.ReactNode
  if (app.page === 'login.html') content = <Login data={data}/>
  else if (app.page === 'catalog.html') content = <Catalog data={data}/>
  else if (app.page === 'image_info.html') content = <ImageDetails data={data}/>
  else if (!data.eventsAllowed) content = <Empty>当前账号无权查看此页面。</Empty>
  else if (app.page === 'statistics.html') content = <Statistics data={data}/>
  else if (app.page === 'purge_log.html') content = <PurgeLog data={data}/>
  else if (app.page === 'options.html') content = <Options data={data}/>
  else content = <Empty>页面不存在。</Empty>
  return <Shell>{content}</Shell>
}
