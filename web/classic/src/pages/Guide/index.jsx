/*
Copyright (C) 2025 QuantumNous

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

import React from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import { IconCopy, IconDownload, IconExternalOpen } from '@douyinfe/semi-icons';
import { copy, showSuccess } from '../../helpers';

const { Text, Title } = Typography;

const baseUrl = 'http://182.92.166.143:3200';
const hapiHubUrl = 'http://182.92.166.143:3006';
const model = 'DSv4-flash';
const assetBase = '/assets/guide';

const mappedSettings = `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "替换为 token-hub 复制的密钥",
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0",
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": 1
  },
  "includeCoAuthoredBy": false
}`;

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
}`;

const ccSwitchAnthropicConfig = `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "通过 cc-switch 配置的 Anthropic 协议密钥",
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0",
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": 1
  },
  "includeCoAuthoredBy": false
}`;

const steps = [
  { id: 'install', label: '安装 Claude Code' },
  { id: 'token', label: '获取 token-hub 密钥' },
  { id: 'settings', label: '直接配置 settings.json' },
  { id: 'cc-anthropic', label: 'cc-switch：Anthropic 协议' },
  { id: 'cc-openai', label: 'cc-switch：OpenAI 协议' },
  { id: 'route', label: '开启本地代理/路由' },
  { id: 'verify', label: '启动验证' },
  { id: 'hapi', label: '可选：HAPI 远程接管' },
];

function CodeBlock({ value }) {
  const handleCopy = async () => {
    if (await copy(value)) {
      showSuccess('Copied');
    }
  };

  return (
    <div className='overflow-hidden rounded-lg border border-semi-color-border bg-semi-color-bg-1'>
      <div className='flex justify-end border-b border-semi-color-border px-3 py-2'>
        <Button
          size='small'
          type='tertiary'
          icon={<IconCopy />}
          onClick={handleCopy}
        >
          复制
        </Button>
      </div>
      <pre className='m-0 overflow-x-auto p-4 text-sm leading-6'>
        <code>{value}</code>
      </pre>
    </div>
  );
}

function GuideImage({ src, alt, className = '' }) {
  return (
    <img
      src={`${assetBase}/${src}`}
      alt={alt}
      className={`w-full rounded-lg border border-semi-color-border bg-semi-color-bg-0 object-contain ${className}`}
      loading='lazy'
    />
  );
}

function Section({ id, title, children }) {
  return (
    <section
      id={id}
      className='scroll-mt-24 border-b border-semi-color-border pb-10'
    >
      <Title heading={2} className='!mb-5 !text-2xl'>
        {title}
      </Title>
      <div className='space-y-5 text-base leading-7'>{children}</div>
    </section>
  );
}

function Tip({ children }) {
  return (
    <div className='rounded-lg border border-semi-color-primary bg-semi-color-primary-light-default p-4 text-sm leading-6'>
      {children}
    </div>
  );
}

export default function Guide() {
  return (
    <div className='h-full overflow-y-auto bg-semi-color-bg-0'>
      <div className='mx-auto flex max-w-7xl gap-8 px-4 pb-8 pt-20 lg:px-6 lg:pt-24'>
        <aside className='hidden w-64 shrink-0 lg:block'>
          <div className='sticky top-24 space-y-3'>
            <Text type='secondary' strong>
              引导目录
            </Text>
            <nav className='space-y-1 text-sm'>
              {steps.map((step) => (
                <a
                  key={step.id}
                  href={`#${step.id}`}
                  className='block rounded-md px-3 py-2 text-semi-color-text-1 hover:bg-semi-color-fill-0 hover:text-semi-color-primary'
                >
                  {step.label}
                </a>
              ))}
            </nav>
          </div>
        </aside>

        <main className='min-w-0 flex-1 space-y-10'>
          <header className='space-y-5'>
            <div className='space-y-3'>
              <Text type='tertiary'>token-hub / Claude Code 新手引导</Text>
              <Title heading={1} className='!my-0 !text-4xl'>
                使用 token-hub 接入 Claude Code
              </Title>
              <p className='max-w-3xl text-base leading-7 text-semi-color-text-1'>
                本页面说明如何安装 Claude Code、从 token-hub 获取密钥，并通过
                settings.json 或 cc-switch 完成模型配置。默认服务地址为{' '}
                <code>{baseUrl}</code>，默认模型为 <code>{model}</code>。
              </p>
              <div className='flex flex-wrap gap-3'>
                <Button
                  theme='solid'
                  type='primary'
                  icon={<IconDownload />}
                  onClick={() => {
                    window.location.href = `${assetBase}/token-hub-claude-code配置指南-新手版.pdf`;
                  }}
                >
                  下载 PDF
                </Button>
                <Button
                  type='tertiary'
                  icon={<IconExternalOpen />}
                  onClick={() => {
                    window.open(
                      'https://docs.anthropic.com/en/docs/claude-code',
                      '_blank',
                      'noopener,noreferrer',
                    );
                  }}
                >
                  Claude Code 文档
                </Button>
              </div>
            </div>
          </header>

          <Section id='install' title='1. 安装 Claude Code'>
            <p>
              Claude Code 在终端中运行。第一次使用前，需要先确认本机已经安装
              Node.js 和 npm。
            </p>
            <CodeBlock value={'node -v\nnpm -v'} />
            <p>如果能看到版本号，继续安装 Claude Code：</p>
            <CodeBlock value='npm install -g @anthropic-ai/claude-code' />
            <p>安装完成后验证命令是否可用：</p>
            <CodeBlock value='claude --version' />
            <Tip>
              Windows 用户建议使用 PowerShell 或 Git Bash。安装后如果提示{' '}
              <code>claude</code> 命令不存在，关闭并重新打开终端再试。
            </Tip>
          </Section>

          <Section id='token' title='2. 从 token-hub 获取密钥'>
            <p>进入 New API 控制台，打开左侧「令牌管理」，点击「添加令牌」。</p>
            <GuideImage
              src='token-list-empty.png'
              alt='令牌管理页面添加令牌'
            />
            <p>
              创建令牌时选择令牌分组。用于 Claude Code 时，推荐选择映射到 cc
              的分组，例如「海口_deepseek机房-映射cc」。
            </p>
            <GuideImage src='token-create.png' alt='创建令牌并选择分组' />
            <p>
              提交后回到列表，点击密钥右侧复制按钮。复制出的密钥用于后续{' '}
              <code>ANTHROPIC_AUTH_TOKEN</code> 或 cc-switch 的 API Key。
            </p>
            <GuideImage src='token-copy.png' alt='复制令牌密钥' />
            <Tip>
              优先使用页面复制按钮，不要手动重打密钥，避免把数字 <code>0</code>{' '}
              看成字母 <code>O</code>。
            </Tip>
          </Section>

          <Section id='settings' title='3. 直接配置 ~/.claude/settings.json'>
            <p>
              不使用 cc-switch 时，直接编辑 Claude Code 的配置文件。Linux /
              macOS 路径是 <code>~/.claude/settings.json</code>，Windows
              PowerShell 路径通常是{' '}
              <code>$env:USERPROFILE\.claude\settings.json</code>。
            </p>
            <CodeBlock
              value={'mkdir -p ~/.claude\nnano ~/.claude/settings.json'}
            />
            <Title heading={3} className='!mb-0 !text-lg'>
              使用 cc 映射令牌
            </Title>
            <p>如果 token-hub 分组已经做了 cc 映射，推荐复制下面的最小配置：</p>
            <CodeBlock value={mappedSettings} />
            <GuideImage
              src='settings-mapped.png'
              alt='settings.json 使用 cc 映射配置'
            />
            <Title heading={3} className='!mb-0 !text-lg'>
              直接指定 DSv4-flash
            </Title>
            <p>如果没有使用 cc 映射，或者希望明确指定模型，使用下面的完整配置：</p>
            <CodeBlock value={directModelSettings} />
            <GuideImage
              src='settings-full.png'
              alt='settings.json 完整指定模型配置'
            />
          </Section>

          <Section id='cc-anthropic' title='4. cc-switch：Anthropic 原生协议'>
            <p>
              在 cc-switch 新建或编辑供应商。API Key 填 token-hub
              密钥，请求地址填 <code>{baseUrl}</code>，不要追加 <code>/v1</code>。
            </p>
            <GuideImage
              src='cc-anthropic.png'
              alt='cc-switch Anthropic 协议配置'
            />
            <CodeBlock value={ccSwitchAnthropicConfig} />
          </Section>

          <Section id='cc-openai' title='5. cc-switch：OpenAI Chat Completions'>
            <p>
              使用 OpenAI 协议时，需要展开「高级选项」，API 格式选择 OpenAI Chat
              Completions。Sonnet、Opus、Fable、Haiku 的实际请求模型都填写为{' '}
              <code>{model}</code>，默认兜底模型也填写 <code>{model}</code>。
            </p>
            <GuideImage
              src='cc-openai.png'
              alt='cc-switch OpenAI Chat Completions 配置'
            />
            <Tip>
              这种方式依赖 cc-switch 的本地代理/路由，把 Claude Code 或 Codex
              请求转发到对应供应商。
            </Tip>
          </Section>

          <Section id='route' title='6. 开启本地代理/路由模式'>
            <p>
              使用 OpenAI Chat Completions 方式时，必须在 cc-switch 中打开本地
              路由/代理模式，并启用对应应用。
            </p>
            <div className='grid gap-4 xl:grid-cols-[2fr_1fr]'>
              <GuideImage src='cc-route.png' alt='cc-switch 路由设置' />
              <GuideImage
                src='cc-switch-toggle.png'
                alt='cc-switch 顶部代理开关'
                className='max-h-40'
              />
            </div>
            <CodeBlock
              value={`本地路由服务地址示例：\nhttp://0.0.0.0:15721\n\n请求链路：\nClaude Code / Codex -> cc-switch 本地路由 -> token-hub -> MASS / DeepSeek`}
            />
          </Section>

          <Section id='verify' title='7. 启动与验证'>
            <ol className='list-decimal space-y-2 pl-5'>
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

          <Section id='hapi' title='可选篇章：使用 HAPI 远程接管 Claude Code'>
            <p>
              HAPI 是远程接管通道，不替代前面的 Claude Code 模型配置。正常本地使用时，
              仍然按第 3 节的 <code>settings.json</code> 或第 4-6 节的 cc-switch
              方式完成配置；需要远程接管时，再用 <code>hapi</code> 唤起本机
              Claude Code 并连接到 HAPI Hub。
            </p>
            <p>
              在令牌列表右侧点击「HAPI」下拉菜单。这里有两个需要用到的内容：
            </p>
            <ul className='list-disc space-y-2 pl-5'>
              <li>
                「复制 HAPI 初始化脚本」用于配置本机 HAPI 连接信息，macOS/Linux
                和 Windows 按自己的系统选择。
              </li>
              <li>
                「复制 HAPI 令牌」用于登录 HAPI Web 页面，远程接管时需要粘贴这个令牌。
              </li>
            </ul>
            <GuideImage
              src='hapi-token-menu.png'
              alt='令牌列表中的 HAPI 操作菜单'
              className='max-w-sm'
            />
            <p>
              把复制出的命令粘贴到本机终端执行一次。脚本会请求 token-hub
              获取当前令牌对应的 HAPI 连接信息，并写入本机{' '}
              <code>~/.hapi/settings.json</code>。
            </p>
            <GuideImage
              src='hapi-setup-command.png'
              alt='执行 HAPI 初始化脚本'
            />
            <CodeBlock value='hapi' />
            <p>
              初始化成功后，在项目目录运行 <code>hapi</code>。它会启动本机 Claude
              Code 并把当前机器接入 HAPI Hub；以后重启电脑或重新接入时，只要继续使用
              同一个 token-hub 令牌生成的 HAPI 配置，就会回到同一个 HAPI namespace。
            </p>
            <Tip>
              默认脚本写入 <code>~/.hapi/settings.json</code>。如果你手动指定过{' '}
              <code>HAPI_HOME</code>，后续每次运行 <code>hapi</code> 都要使用同一个{' '}
              <code>HAPI_HOME</code>，否则会读到另一套本地配置。
            </Tip>
            <p>
              Web 端登录时，打开 <code>{hapiHubUrl}/sessions</code>。回到令牌列表的
              「HAPI」下拉菜单，点击「复制 HAPI 令牌」，把复制出的令牌粘贴到登录页。
            </p>
            <GuideImage src='hapi-login.png' alt='HAPI Web 登录页' />
            <p>
              登录后选择本机机器和会话，即可在浏览器里继续远程操作 Claude Code。
              本机通过 <code>hapi</code> 启动后，页面会显示对应的 session。
            </p>
            <GuideImage src='hapi-session.png' alt='HAPI 远程会话页面' />
            <Tip>
              HAPI namespace 用于把不同 token-hub 令牌的机器和会话分开，方便恢复原来的工作状态。
              它不是高强度安全边界，不要把 HAPI 令牌或初始化脚本公开发送给无关人员。
            </Tip>
          </Section>
        </main>
      </div>
    </div>
  );
}
