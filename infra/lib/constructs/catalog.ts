import { RemovalPolicy } from 'aws-cdk-lib/core';
import { AttributeType, TableV2 } from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';

/**
 * Single-table catalog of a user's drive. Everything a user owns lives in
 * their partition, and each question the application asks is one query on a
 * sort-key prefix:
 *
 *   pk = user#<sub>   sk = #file#<object key>            my files
 *   pk = user#<sub>   sk = #index#<label>#<object key>   files with a label
 *   pk = user#<sub>   sk = #label#<label>                the labels I have
 *
 * The table is shaped for reads: the pipeline writes one small item per
 * answer when a photo is uploaded, so a search never reads a growing list.
 * Every item is keyed by what it describes, which makes the pipeline's
 * writes safe to retry and replay.
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
