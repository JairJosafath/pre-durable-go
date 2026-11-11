import { RemovalPolicy } from 'aws-cdk-lib/core';
import { AttributeType, TableV2 } from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';

/**
 * Single-table catalog of everything a user uploaded.
 *
 *   pk = user#<sub>   sk = #file#<object key>             one item per upload
 *   pk = user#<sub>   sk = #index#<label>#<object key>    one item per detected label
 *
 * Label items are keyed by label and object, so writing one twice leaves a
 * single item. That makes the indexing step of the durable function safe to
 * retry and replay.
 */
export class Catalog extends Construct {
  readonly table: TableV2;

  constructor(scope: Construct, id: string) {
    super(scope, id);

    this.table = new TableV2(this, 'Table', {
      partitionKey: { name: 'pk', type: AttributeType.STRING },
      sortKey: { name: 'sk', type: AttributeType.STRING },
      removalPolicy: RemovalPolicy.DESTROY,
    });
  }
}
