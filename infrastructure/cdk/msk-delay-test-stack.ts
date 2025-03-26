import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as msk from 'aws-cdk-lib/aws-msk';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import { Construct } from 'constructs';

export class MskDelayTestStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Create VPC
    const vpc = new ec2.Vpc(this, 'MskTestVpc', {
      maxAzs: 3,
      natGateways: 1,
    });

    // Create security group for MSK
    const mskSecurityGroup = new ec2.SecurityGroup(this, 'MskSecurityGroup', {
      vpc,
      description: 'Security group for MSK cluster',
      allowAllOutbound: true,
    });

    // Allow inbound traffic on Kafka ports
    mskSecurityGroup.addIngressRule(
      ec2.Peer.ipv4(vpc.vpcCidrBlock),
      ec2.Port.tcp(9092),
      'Allow Kafka plaintext traffic'
    );
    mskSecurityGroup.addIngressRule(
      ec2.Peer.ipv4(vpc.vpcCidrBlock),
      ec2.Port.tcp(9094),
      'Allow Kafka TLS traffic'
    );
    mskSecurityGroup.addIngressRule(
      ec2.Peer.ipv4(vpc.vpcCidrBlock),
      ec2.Port.tcp(2181),
      'Allow Zookeeper traffic'
    );

    // Create CloudWatch log group for MSK broker logs
    const mskLogGroup = new logs.LogGroup(this, 'MskBrokerLogs', {
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Create MSK cluster
    const mskCluster = new msk.CfnCluster(this, 'MskCluster', {
      clusterName: 'msk-delay-test-cluster',
      kafkaVersion: '3.5.1',
      numberOfBrokerNodes: 3,
      brokerNodeGroupInfo: {
        instanceType: 'kafka.m5.large',
        clientSubnets: vpc.privateSubnets.map(subnet => subnet.subnetId),
        securityGroups: [mskSecurityGroup.securityGroupId],
        storageInfo: {
          ebsStorageInfo: {
            volumeSize: 100,
          },
        },
      },
      encryptionInfo: {
        encryptionInTransit: {
          clientBroker: 'PLAINTEXT',
          inCluster: true,
        },
      },
      loggingInfo: {
        brokerLogs: {
          cloudWatchLogs: {
            enabled: true,
            logGroup: mskLogGroup.logGroupName,
          },
        },
      },
      openMonitoring: {
        prometheus: {
          jmxExporter: {
            enabledInBroker: true,
          },
          nodeExporter: {
            enabledInBroker: true,
          },
        },
      },
      configurationInfo: {
        arn: new msk.CfnConfiguration(this, 'MskConfiguration', {
          name: 'msk-delay-test-config',
          serverProperties: [
            'auto.create.topics.enable=true',
            'default.replication.factor=3',
            'min.insync.replicas=2',
            'num.partitions=50',
            'log.retention.hours=24',
            'group.initial.rebalance.delay.ms=0',
          ].join('\\n'),
        }).attrArn,
        revision: 1,
      },
    });

    // Create security group for EC2 instances
    const ec2SecurityGroup = new ec2.SecurityGroup(this, 'Ec2SecurityGroup', {
      vpc,
      description: 'Security group for EC2 instances',
      allowAllOutbound: true,
    });

    // Allow SSH access
    ec2SecurityGroup.addIngressRule(
      ec2.Peer.anyIpv4(),
      ec2.Port.tcp(22),
      'Allow SSH access'
    );

    // Create IAM role for EC2 instances
    const ec2Role = new iam.Role(this, 'Ec2Role', {
      assumedBy: new iam.ServicePrincipal('ec2.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSSMManagedInstanceCore'),
      ],
    });

    // Add MSK permissions
    ec2Role.addToPolicy(new iam.PolicyStatement({
      actions: [
        'kafka:DescribeCluster',
        'kafka:GetBootstrapBrokers',
        'kafka:ListClusters',
        'kafka:DescribeClusterV2',
        'kafka:DescribeTopic',
        'kafka:ListTopics',
        'kafka:CreateTopic',
        'kafka:DeleteTopic',
      ],
      resources: ['*'],
    }));

    // Add CloudWatch permissions
    ec2Role.addToPolicy(new iam.PolicyStatement({
      actions: [
        'logs:CreateLogGroup',
        'logs:CreateLogStream',
        'logs:PutLogEvents',
        'logs:DescribeLogStreams',
      ],
      resources: ['*'],
    }));

    // Create producer EC2 instance
    const producerInstance = new ec2.Instance(this, 'ProducerInstance', {
      vpc,
      instanceType: ec2.InstanceType.of(ec2.InstanceClass.T3, ec2.InstanceSize.MEDIUM),
      machineImage: ec2.MachineImage.latestAmazonLinux2023(),
      securityGroup: ec2SecurityGroup,
      role: ec2Role,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PUBLIC,
      },
      keyName: 'msk-delay-test-key', // Make sure to create this key pair in the AWS console
    });

    // Create consumer EC2 instance
    const consumerInstance = new ec2.Instance(this, 'ConsumerInstance', {
      vpc,
      instanceType: ec2.InstanceType.of(ec2.InstanceClass.T3, ec2.InstanceSize.LARGE),
      machineImage: ec2.MachineImage.latestAmazonLinux2023(),
      securityGroup: ec2SecurityGroup,
      role: ec2Role,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PUBLIC,
      },
      keyName: 'msk-delay-test-key', // Make sure to create this key pair in the AWS console
    });

    // Create CloudWatch dashboard for monitoring
    const dashboard = new cloudwatch.Dashboard(this, 'MskDelayTestDashboard', {
      dashboardName: 'MSK-Delay-Test-Dashboard',
    });

    // Add MSK metrics to dashboard
    dashboard.addWidgets(
      new cloudwatch.GraphWidget({
        title: 'MSK - CPU Utilization',
        left: [
          new cloudwatch.Metric({
            namespace: 'AWS/Kafka',
            metricName: 'CPUUtilization',
            dimensionsMap: {
              'Cluster Name': mskCluster.clusterName,
            },
            statistic: 'Average',
            period: cdk.Duration.minutes(1),
          }),
        ],
      }),
      new cloudwatch.GraphWidget({
        title: 'MSK - Memory Used',
        left: [
          new cloudwatch.Metric({
            namespace: 'AWS/Kafka',
            metricName: 'MemoryUsed',
            dimensionsMap: {
              'Cluster Name': mskCluster.clusterName,
            },
            statistic: 'Average',
            period: cdk.Duration.minutes(1),
          }),
        ],
      }),
      new cloudwatch.GraphWidget({
        title: 'MSK - Disk Used',
        left: [
          new cloudwatch.Metric({
            namespace: 'AWS/Kafka',
            metricName: 'KafkaDataLogsDiskUsed',
            dimensionsMap: {
              'Cluster Name': mskCluster.clusterName,
            },
            statistic: 'Average',
            period: cdk.Duration.minutes(1),
          }),
        ],
      }),
      new cloudwatch.GraphWidget({
        title: 'MSK - Consumer Group Rebalances',
        left: [
          new cloudwatch.MathExpression({
            expression: 'SEARCH(\\'{AWS/Kafka,Cluster Name,Topic} MetricName="GroupRebalances"\\', \'Sum\', 3600)',
            label: 'Consumer Group Rebalances per Hour',
            period: cdk.Duration.hours(1),
          }),
        ],
      })
    );

    // Output the MSK bootstrap brokers
    new cdk.CfnOutput(this, 'MskBootstrapBrokers', {
      value: cdk.Fn.getAtt(mskCluster.logicalId, 'BootstrapBrokers').toString(),
      description: 'MSK Bootstrap Brokers',
      exportName: 'MskBootstrapBrokers',
    });

    // Output the producer instance ID
    new cdk.CfnOutput(this, 'ProducerInstanceId', {
      value: producerInstance.instanceId,
      description: 'Producer EC2 Instance ID',
      exportName: 'ProducerInstanceId',
    });

    // Output the consumer instance ID
    new cdk.CfnOutput(this, 'ConsumerInstanceId', {
      value: consumerInstance.instanceId,
      description: 'Consumer EC2 Instance ID',
      exportName: 'ConsumerInstanceId',
    });
  }
}
