import * as core from '@actions/core'
import * as github from '@actions/github'
import * as httpm from '@actions/http-client'
import {WebhookPayload} from '@actions/github/lib/interfaces'
import type {WorkflowRunCompletedEvent} from '@octokit/webhooks-types'

const http = new httpm.HttpClient('client')

async function getAccessToken(
  clientId: string,
  clientSecret: string
): Promise<string> {
  const response = await http.post(
    'https://sso.dynatrace.com/sso/oauth2/token',
    `grant_type=client_credentials&client_id=${clientId}&client_secret=${clientSecret}&scope=storage:events:write`,
    {
      'content-type': 'application/x-www-form-urlencoded'
    }
  )
  const body = JSON.parse(await response.readBody())
  return body.access_token as string
}

function buildCloudEvent(payload: WebhookPayload): unknown {
  const workflowRun = (payload as WorkflowRunCompletedEvent).workflow_run
  return {
    specversion: '1.0',
    id: `${workflowRun.id}`,
    type: 'com.dynatrace.github.workflow.run',
    source: 'dynatrace-workflow-ingester',
    data: {
      ...workflowRun,
      run_duration_ms:
        new Date(workflowRun.updated_at).getTime() -
        new Date(workflowRun.run_started_at).getTime()
    }
  }
}

async function run(): Promise<void> {
  try {
    const clientId = core.getInput('dt-client-id')
    const clientSecret = core.getInput('dt-client-secret')
    const environmentId = core.getInput('dt-environment-id')
    const cloudEvent = buildCloudEvent(github.context.payload)
    const dynatraceAccessToken = await getAccessToken(clientId, clientSecret)
    const response = await http.post(
      `https://${environmentId}.live.dynatrace.com/api/v2/bizevents/ingest`,
      JSON.stringify(cloudEvent),
      {
        'content-type': 'application/cloudevent+json',
        authorization: `Bearer ${dynatraceAccessToken}`
      }
    )
    core.info(await response.readBody())
  } catch (error) {
    if (error instanceof Error) core.setFailed(error.message)
  }
}

run()
