import {
  AuthorizationType,
  AwsIntegration,
  CognitoUserPoolsAuthorizer,
  Cors,
  IntegrationResponse,
  LambdaIntegration,
  MethodOptions,
  MethodResponse,
  RestApi,
} from 'aws-cdk-lib/aws-apigateway';
import { UserPool, UserPoolClient } from 'aws-cdk-lib/aws-cognito';
import { TableV2 } from 'aws-cdk-lib/aws-dynamodb';
import { PolicyStatement, Role, ServicePrincipal } from 'aws-cdk-lib/aws-iam';
import { Alias } from 'aws-cdk-lib/aws-lambda';
import { Bucket } from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';
import { goLambda } from './go-function';

export interface PrimeDriveApiProps {
  readonly userPool: UserPool;
  readonly client: UserPoolClient;
  readonly uploads: Bucket;
  readonly thumbnails: Bucket;
  readonly table: TableV2;
  /** The durable function whose executions wait for review decisions. */
  readonly workflow: Alias;
}

const corsHeaders = {
  'method.response.header.Access-Control-Allow-Origin': "'*'",
};

/** Responses for direct service integrations, with CORS headers for the browser. */
const integrationResponses: IntegrationResponse[] = [
  {
    statusCode: '200',
    responseParameters: {
      ...corsHeaders,
      'method.response.header.Content-Type': 'integration.response.header.Content-Type',
    },
  },
  { statusCode: '400', selectionPattern: '4\\d{2}', responseParameters: corsHeaders },
  { statusCode: '500', selectionPattern: '5\\d{2}', responseParameters: corsHeaders },
];

const methodResponses: MethodResponse[] = [
  {
    statusCode: '200',
    responseParameters: {
      'method.response.header.Access-Control-Allow-Origin': true,
      'method.response.header.Content-Type': true,
    },
  },
  { statusCode: '400', responseParameters: { 'method.response.header.Access-Control-Allow-Origin': true } },
  { statusCode: '500', responseParameters: { 'method.response.header.Access-Control-Allow-Origin': true } },
];

/** Items per page for catalog queries. */
const PAGE_SIZE = 24;

/** A query-string parameter as a value safe to embed in a mapping template. */
const param = (name: string, transform = '') =>
  `$util.escapeJavaScript($input.params('${name}')${transform}).replaceAll("\\'","")`;

/** The signed-in user's Cognito subject, as a value safe to embed in a mapping template. */
const SUB = `$util.escapeJavaScript($context.authorizer.claims.sub).replaceAll("\\'","")`;

/**
 * REST API for the web frontend.
 *
 *   POST   /auth/{login,signup,confirm}   Cognito, called directly by API Gateway
 *   GET    /files                         the user's uploads (DynamoDB query)
 *   GET    /query?index=<label>           uploads by label (DynamoDB query)
 *   GET    /labels?prefix=<text>          the user's labels, for suggestions (DynamoDB query)
 *   GET    /object/{key+}                 thumbnail of an upload (S3)
 *   DELETE /object/{key+}                 delete an upload (S3)
 *   GET    /presigned?action=&key=        presigned URL for an original (Lambda)
 *   POST   /reviews                       approve or reject an upload (Lambda)
 */
export class PrimeDriveApi extends Construct {
  readonly api: RestApi;

  private readonly serviceRole: Role;
  private readonly signedIn: MethodOptions;

  constructor(scope: Construct, id: string, props: PrimeDriveApiProps) {
    super(scope, id);

    this.api = new RestApi(this, 'RestApi', {
      restApiName: 'PrimeDrive',
      defaultCorsPreflightOptions: {
        allowOrigins: Cors.ALL_ORIGINS,
        allowMethods: Cors.ALL_METHODS,
      },
      binaryMediaTypes: ['image/*', 'application/octet-stream'],
    });

    // Role API Gateway assumes for the integrations that call AWS services directly.
    this.serviceRole = new Role(this, 'ServiceIntegrationRole', {
      assumedBy: new ServicePrincipal('apigateway.amazonaws.com'),
      description: 'Lets API Gateway call Cognito, DynamoDB and S3 on behalf of a signed-in user',
    });

    this.signedIn = {
      authorizationType: AuthorizationType.COGNITO,
      authorizer: new CognitoUserPoolsAuthorizer(this, 'Authorizer', { cognitoUserPools: [props.userPool] }),
      methodResponses,
    };

    this.addAuthRoutes(props.client);
    this.addCatalogRoutes(props.table);
    this.addObjectRoutes(props.uploads, props.thumbnails);
    this.addPresignRoute(props.uploads);
    this.addReviewRoute(props.table, props.workflow);
  }

  private addAuthRoutes(client: UserPoolClient): void {
    const auth = this.api.root.addResource('auth');
    const clientId = client.userPoolClientId;

    const routes: Record<string, { action: string; template: string }> = {
      login: {
        action: 'InitiateAuth',
        template: `{
          "AuthFlow": "USER_PASSWORD_AUTH",
          "ClientId": "${clientId}",
          "AuthParameters": { "USERNAME": $input.json('$.username'), "PASSWORD": $input.json('$.password') }
        }`,
      },
      signup: {
        action: 'SignUp',
        template: `{
          "ClientId": "${clientId}",
          "Username": $input.json('$.username'),
          "Password": $input.json('$.password'),
          "UserAttributes": [
            { "Name": "custom:subscription", "Value": "free" },
            { "Name": "email", "Value": $input.json('$.email') }
          ]
        }`,
      },
      confirm: {
        action: 'ConfirmSignUp',
        template: `{
          "ClientId": "${clientId}",
          "Username": $input.json('$.username'),
          "ConfirmationCode": $input.json('$.code')
        }`,
      },
    };

    for (const [name, route] of Object.entries(routes)) {
      const integration = new AwsIntegration({
        service: 'cognito-idp',
        action: route.action,
        integrationHttpMethod: 'POST',
        options: {
          credentialsRole: this.serviceRole,
          requestTemplates: { 'application/json': route.template },
          integrationResponses,
        },
      });
      auth.addResource(name).addMethod('POST', integration, { methodResponses });
    }
  }

  private addCatalogRoutes(table: TableV2): void {
    table.grantReadData(this.serviceRole);

    // Every route is one query on the user's partition; they differ only in
    // the sort-key prefix. A page limit keeps the cost of a read flat however
    // large a drive grows. The client asks for the next page with the sort
    // key of the last item it received; the partition key always comes from
    // the caller's token, so a page can never start in another user's data.
    const query = (sortKeyPrefix: string) =>
      new AwsIntegration({
        service: 'dynamodb',
        action: 'Query',
        integrationHttpMethod: 'POST',
        options: {
          credentialsRole: this.serviceRole,
          requestTemplates: {
            'application/json': `#set($next = $input.params('next'))
            {
              "TableName": "${table.tableName}",
              "Limit": ${PAGE_SIZE},
              "KeyConditionExpression": "#pk = :pk AND begins_with(#sk, :sk)",
              "ExpressionAttributeNames": { "#pk": "pk", "#sk": "sk" },
              "ExpressionAttributeValues": {
                ":pk": { "S": "user#${SUB}" },
                ":sk": { "S": "${sortKeyPrefix}" }
              }
              #if($next != "")
              ,"ExclusiveStartKey": {
                "pk": { "S": "user#${SUB}" },
                "sk": { "S": "${param('next')}" }
              }
              #end
            }`,
          },
          integrationResponses,
        },
      });

    const paged = { 'method.request.querystring.next': false };
    const lowercase = '.toLowerCase()';

    this.api.root.addResource('files').addMethod('GET', query('#file#'), {
      ...this.signedIn,
      requestParameters: paged,
    });

    this.api.root.addResource('query').addMethod('GET', query(`#index#${param('index', lowercase)}`), {
      ...this.signedIn,
      requestParameters: { ...paged, 'method.request.querystring.index': true },
    });

    this.api.root.addResource('labels').addMethod('GET', query(`#label#${param('prefix', lowercase)}`), {
      ...this.signedIn,
      requestParameters: { ...paged, 'method.request.querystring.prefix': false },
    });
  }

  private addObjectRoutes(uploads: Bucket, thumbnails: Bucket): void {
    thumbnails.grantRead(this.serviceRole);
    uploads.grantDelete(this.serviceRole);

    // A user can only ever reach keys under their own users/<sub>/ prefix.
    const object = (method: string, bucket: Bucket) =>
      new AwsIntegration({
        service: 's3',
        integrationHttpMethod: method,
        path: `${bucket.bucketName}/users/{sub}/{key}`,
        options: {
          credentialsRole: this.serviceRole,
          requestParameters: {
            'integration.request.path.sub': 'context.authorizer.claims.sub',
            'integration.request.path.key': 'method.request.path.key',
          },
          integrationResponses,
        },
      });

    const resource = this.api.root.addResource('object').addResource('{key+}');
    const options = { ...this.signedIn, requestParameters: { 'method.request.path.key': true } };
    resource.addMethod('GET', object('GET', thumbnails), options);
    resource.addMethod('DELETE', object('DELETE', uploads), options);
  }

  private addPresignRoute(uploads: Bucket): void {
    const presigner = goLambda(this, 'Presigner', 'presigner', {
      description: 'Creates presigned URLs to upload and download originals',
      environment: { UPLOAD_BUCKET: uploads.bucketName },
    });
    uploads.grantReadWrite(presigner);

    this.api.root.addResource('presigned').addMethod('GET', new LambdaIntegration(presigner), this.signedIn);
  }

  private addReviewRoute(table: TableV2, workflow: Alias): void {
    const reviewer = goLambda(this, 'Reviewer', 'reviewer', {
      description: 'Sends a review decision to the durable execution waiting for it',
      environment: { TABLE_NAME: table.tableName },
    });
    table.grantReadData(reviewer);

    // Completing a callback resumes the suspended execution. The grant covers
    // the workflow function and all of its versions and aliases.
    const workflowArn = workflow.lambda.functionArn;
    reviewer.addToRolePolicy(
      new PolicyStatement({
        actions: ['lambda:SendDurableExecutionCallbackSuccess', 'lambda:SendDurableExecutionCallbackFailure'],
        resources: [workflowArn, `${workflowArn}:*`],
      }),
    );

    this.api.root.addResource('reviews').addMethod('POST', new LambdaIntegration(reviewer), this.signedIn);
  }
}
