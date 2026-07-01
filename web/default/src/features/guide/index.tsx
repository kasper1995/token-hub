/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { CheckCircle2, Download, ExternalLink } from 'lucide-react'
import { CopyButton } from '@/components/copy-button'
import { PublicLayout } from '@/components/layout'

const baseUrl = 'http://182.92.166.143:3200'
const model = 'DSv4-flash'

const mappedSettings = `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "替换为 token-hub 复制的密钥",
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0",
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": 1
  },
  "includeCoAuthoredBy": false
}`

const directModelSettings = `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "替换为 token-hub 复制的密钥",
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "ANTHROPIC_MODEL": "${model}",
    "ANTHROPIC_DEFAULT_FABLE_MODEL": "${model}",
    "ANTHROPIC_DEFAULT_FABLE_MODEL_NAME": "${model}",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "${model}",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME": "${model}",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "${model}",
    "ANTHROPIC_DEFAULT_OPUS_MODEL_NAME": "${model}",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "${model}",
    "ANTHROPIC_DEFAULT_SONNET_MODEL_NAME": "${model}",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0",
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": 1
  },
  "includeCoAuthoredBy": false
}`

const ccSwitchAnthropicConfig = `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "通过 cc-switch 配置的 Anthropic 协议密钥",
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0",
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": 1
  },
  "includeCoAuthoredBy": false
}`

const steps = [
  { id: 'install', label: '安装 Claude Code' },
  { id: 'token', label: '获取 token-hub 密钥' },
  { id: 'settings', label: '直接配置 settings.json' },
  { id: 'cc-anthropic', label: 'cc-switch：Anthropic 协议' },
  { id: 'cc-openai', label: 'cc-switch：OpenAI 协议' },
  { id: 'route', label: '开启本地代理/路由' },
  { id: 'verify', label: '启动验证' },
]

function CodeBlock({ value }: { value: string }) {
  return (
    <div className='bg-card overflow-hidden rounded-lg border'>
      <div className='border-b px-3 py-2 text-right'>
        <CopyButton
          value={value}
          variant='outline'
          size='sm'
          tooltip='复制配置'
          successTooltip='已复制'
        >
          复制
        </CopyButton>
      </div>
      <pre className='overflow-x-auto p-4 text-sm leading-6'>
        <code>{value}</code>
      </pre>
    </div>
  )
}

function GuideImage({
  src,
  alt,
  className,
}: {
  src: string
  alt: string
  className?: string
}) {
  return (
    <img
      src={src}
      alt={alt}
      className={`bg-background w-full rounded-lg border object-contain ${className ?? ''}`}
      loading='lazy'
    />
  )
}

function Section({
  id,
  title,
  children,
}: {
  id: string
  title: string
  children: React.ReactNode
}) {
  return (
    <section id={id} className='scroll-mt-24 space-y-5 border-b pb-10'>
      <h2 className='text-2xl font-semibold tracking-tight'>{title}</h2>
      {children}
    </section>
  )
}

function Tip({ children }: { children: React.ReactNode }) {
  return (
    <div className='bg-muted/50 flex gap-3 rounded-lg border p-4 text-sm leading-6'>
      <CheckCircle2 className='text-primary mt-0.5 h-5 w-5 shrink-0' />
      <div>{children}</div>
    </div>
  )
}

export function Guide() {
  return (
    <PublicLayout>
      <div className='mx-auto flex max-w-7xl gap-8 px-4 py-8 lg:px-6'>
        <aside className='hidden w-64 shrink-0 lg:block'>
          <div className='sticky top-20 space-y-3'>
            <div className='text-muted-foreground text-sm font-medium'>
              引导目录
            </div>
            <nav className='space-y-1 text-sm'>
              {steps.map((step) => (
                <a
                  key={step.id}
                  href={`#${step.id}`}
                  className='hover:bg-muted block rounded-md px-3 py-2'
                >
                  {step.label}
                </a>
              ))}
            </nav>
          </div>
        </aside>

        <main className='min-w-0 flex-1 space-y-10'>
          <header className='space-y-5'>
            <GuideImage
              src='/guide/top-nav-guide-target.png'
              alt='New API 顶部导航中的引导入口'
              className='max-h-20'
            />
            <div className='space-y-3'>
              <div className='text-primary text-sm font-medium'>
                token-hub / Claude Code 新手引导
              </div>
              <h1 className='text-4xl font-bold tracking-tight'>
                使用 token-hub 接入 Claude Code
              </h1>
              <p className='text-muted-foreground max-w-3xl text-base leading-7'>
                本页面说明如何安装 Claude Code、从 token-hub 获取密钥，并通过
                settings.json 或 cc-switch 完成模型配置。默认服务地址为{' '}
                <code>{baseUrl}</code>，默认模型为 <code>{model}</code>。
              </p>
              <div className='flex flex-wrap gap-3'>
                <a
                  href='/guide/token-hub-claude-code配置指南-新手版.pdf'
                  className='bg-primary text-primary-foreground inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium'
                  download
                >
                  <Download className='h-4 w-4' />
                  下载 PDF
                </a>
                <a
                  href='https://docs.anthropic.com/en/docs/claude-code'
                  target='_blank'
                  rel='noreferrer'
                  className='inline-flex items-center gap-2 rounded-md border px-4 py-2 text-sm font-medium'
                >
                  <ExternalLink className='h-4 w-4' />
                  Claude Code 文档
                </a>
              </div>
            </div>
          </header>

          <Section id='install' title='1. 安装 Claude Code'>
            <p className='leading-7'>
              Claude Code 在终端中运行。第一次使用前，需要先确认本机已经安装
              Node.js 和 npm。
            </p>
            <CodeBlock value={'node -v\nnpm -v'} />
            <p className='leading-7'>
              如果能看到版本号，继续安装 Claude Code：
            </p>
            <CodeBlock value='npm install -g @anthropic-ai/claude-code' />
            <p className='leading-7'>安装完成后验证命令是否可用：</p>
            <CodeBlock value='claude --version' />
            <Tip>
              Windows 用户建议使用 PowerShell 或 Git Bash。安装后如果提示
              <code>claude</code> 命令不存在，关闭并重新打开终端再试。
            </Tip>
          </Section>

          <Section id='token' title='2. 从 token-hub 获取密钥'>
            <div className='space-y-4'>
              <p className='leading-7'>
                进入 New API 控制台，打开左侧「令牌管理」，点击「添加令牌」。
              </p>
              <GuideImage
                src='/guide/token-list-empty.png'
                alt='令牌管理页面添加令牌'
              />
              <p className='leading-7'>
                创建令牌时选择令牌分组。用于 Claude Code 时，推荐选择映射到 cc
                的分组，例如「海口_deepseek机房-映射cc」。
              </p>
              <GuideImage
                src='/guide/token-create.png'
                alt='创建令牌并选择分组'
              />
              <p className='leading-7'>
                提交后回到列表，点击密钥右侧复制按钮。复制出的密钥用于后续
                <code>ANTHROPIC_AUTH_TOKEN</code> 或 cc-switch 的 API Key。
              </p>
              <GuideImage src='/guide/token-copy.png' alt='复制令牌密钥' />
              <Tip>
                优先使用页面复制按钮，不要手动重打密钥，避免把数字{' '}
                <code>0</code> 看成字母 <code>O</code>。
              </Tip>
            </div>
          </Section>

          <Section id='settings' title='3. 直接配置 ~/.claude/settings.json'>
            <p className='leading-7'>
              不使用 cc-switch 时，直接编辑 Claude Code 的配置文件。Linux /
              macOS 路径是 <code>~/.claude/settings.json</code>，Windows
              PowerShell 路径通常是{' '}
              <code>$env:USERPROFILE\.claude\settings.json</code>。
            </p>
            <CodeBlock
              value={'mkdir -p ~/.claude\nnano ~/.claude/settings.json'}
            />
            <h3 className='text-lg font-semibold'>使用 cc 映射令牌</h3>
            <p className='leading-7'>
              如果 token-hub 分组已经做了 cc 映射，推荐复制下面的最小配置：
            </p>
            <CodeBlock value={mappedSettings} />
            <GuideImage
              src='/guide/settings-mapped.png'
              alt='settings.json 使用 cc 映射配置'
            />
            <h3 className='text-lg font-semibold'>直接指定 DSv4-flash</h3>
            <p className='leading-7'>
              如果没有使用 cc 映射，或者希望明确指定模型，使用下面的完整配置：
            </p>
            <CodeBlock value={directModelSettings} />
            <GuideImage
              src='/guide/settings-full.png'
              alt='settings.json 完整指定模型配置'
            />
          </Section>

          <Section id='cc-anthropic' title='4. cc-switch：Anthropic 原生协议'>
            <p className='leading-7'>
              在 cc-switch 新建或编辑供应商。API Key 填 token-hub
              密钥，请求地址填
              <code>{baseUrl}</code>，不要追加 <code>/v1</code>。
            </p>
            <GuideImage
              src='/guide/cc-anthropic.png'
              alt='cc-switch Anthropic 协议配置'
            />
            <CodeBlock value={ccSwitchAnthropicConfig} />
          </Section>

          <Section id='cc-openai' title='5. cc-switch：OpenAI Chat Completions'>
            <p className='leading-7'>
              使用 OpenAI 协议时，需要展开「高级选项」，API 格式选择 OpenAI Chat
              Completions。Sonnet、Opus、Fable、Haiku 的实际请求模型都填写为
              <code>{model}</code>，默认兜底模型也填写 <code>{model}</code>。
            </p>
            <GuideImage
              src='/guide/cc-openai.png'
              alt='cc-switch OpenAI Chat Completions 配置'
            />
            <Tip>
              这种方式依赖 cc-switch 的本地代理/路由，把 Claude Code 或 Codex
              请求转发到对应供应商。
            </Tip>
          </Section>

          <Section id='route' title='6. 开启本地代理/路由模式'>
            <p className='leading-7'>
              使用 OpenAI Chat Completions 方式时，必须在 cc-switch 中打开本地
              路由/代理模式，并启用对应应用。
            </p>
            <div className='grid gap-4 xl:grid-cols-[2fr_1fr]'>
              <GuideImage src='/guide/cc-route.png' alt='cc-switch 路由设置' />
              <GuideImage
                src='/guide/cc-switch-toggle.png'
                alt='cc-switch 顶部代理开关'
                className='max-h-36'
              />
            </div>
            <CodeBlock
              value={`本地路由服务地址示例：\nhttp://0.0.0.0:15721\n\n请求链路：\nClaude Code / Codex -> cc-switch 本地路由 -> token-hub -> MASS / DeepSeek`}
            />
          </Section>

          <Section id='verify' title='7. 启动与验证'>
            <ol className='list-decimal space-y-2 pl-5 leading-7'>
              <li>
                保存 <code>~/.claude/settings.json</code> 或确认 cc-switch
                当前供应商已选中。
              </li>
              <li>
                如果使用 OpenAI Chat Completions，确认本地代理/路由开关已开启。
              </li>
              <li>关闭当前终端，重新打开。</li>
              <li>
                进入项目目录运行 <code>claude</code>。
              </li>
              <li>
                输入 <code>hi</code> 等简单问题；能返回模型内容即表示配置成功。
              </li>
            </ol>
            <Tip>
              新手推荐路径：先创建「海口_deepseek机房-映射cc」分组令牌，再使用第
              3 节的最小 <code>settings.json</code> 配置。
            </Tip>
          </Section>
        </main>
      </div>
    </PublicLayout>
  )
}
