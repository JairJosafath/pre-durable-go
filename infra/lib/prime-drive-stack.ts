import { CfnOutput, Stack, StackProps } from 'aws-cdk-lib/core';
import { Construct } from 'constructs';
import { Catalog } from './constructs/catalog';
import { Identity } from './constructs/identity';
import { PrimeDriveApi } from './constructs/rest-api';
import { Storage } from './constructs/storage';
import { UploadPipeline } from './constructs/upload-pipeline';

/**
 * PrimeDrive: upload a photo, get it labelled, reviewed, thumbnailed and
 * indexed so it can be found by what is in it.
 */
export class PrimeDriveStack extends Stack {
  constructor(scope: Construct, id: string, props?: StackProps) {
    super(scope, id, props);

    const storage = new Storage(this, 'Storage');
    const catalog = new Catalog(this, 'Catalog');
    const identity = new Identity(this, 'Identity');

    const pipeline = new UploadPipeline(this, 'UploadPipeline', {
      uploads: storage.uploads,
      thumbnails: storage.thumbnails,
      table: catalog.table,
    });

    const api = new PrimeDriveApi(this, 'Api', {
      userPool: identity.userPool,
      client: identity.client,
      uploads: storage.uploads,
      thumbnails: storage.thumbnails,
      table: catalog.table,
      workflow: pipeline.workflow,
    });

    new CfnOutput(this, 'ApiUrl', { value: api.api.url, description: 'NEXT_PUBLIC_ENDPOINT for the frontend' });
    new CfnOutput(this, 'UploadBucket', { value: storage.uploads.bucketName });
    new CfnOutput(this, 'TableName', { value: catalog.table.tableName });
    new CfnOutput(this, 'WorkflowFunction', {
      value: pipeline.workflow.functionArn,
      description: 'Qualified ARN of the durable function, for the cloud test runner',
    });
  }
}
