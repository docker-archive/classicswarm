package mturk_test

var BasicHitResponse = `<?xml version="1.0"?>
<CreateHITResponse><OperationRequest><RequestId>643b794b-66b6-4427-bb8a-4d3df5c9a20e</RequestId></OperationRequest><HIT><Request><IsValid>True</IsValid></Request><HITId>28J4IXKO2L927XKJTHO34OCDNASCDW</HITId><HITTypeId>2XZ7D1X3V0FKQVW7LU51S7PKKGFKDF</HITTypeId></HIT></CreateHITResponse>
`

var SearchHITResponse = `<?xml version="1.0"?>
<SearchHITsResponse><OperationRequest><RequestId>38862d9c-f015-4177-a2d3-924110a9d6f2</RequestId></OperationRequest><SearchHITsResult><Request><IsValid>True</IsValid></Request><NumResults>1</NumResults><TotalNumResults>1</TotalNumResults><PageNumber>1</PageNumber><HIT><HITId>2BU26DG67D1XTE823B3OQ2JF2XWF83</HITId><HITTypeId>22OWJ5OPB0YV6IGL5727KP9U38P5XR</HITTypeId><CreationTime>2011-12-28T19:56:20Z</CreationTime><Title>test hit</Title><Description>please disregard, testing only</Description><HITStatus>Reviewable</HITStatus><MaxAssignments>1</MaxAssignments><Reward><Amount>0.01</Amount><CurrencyCode>USD</CurrencyCode><FormattedPrice>$0.01</FormattedPrice></Reward><AutoApprovalDelayInSeconds>2592000</AutoApprovalDelayInSeconds><Expiration>2011-12-28T19:56:50Z</Expiration><AssignmentDurationInSeconds>30</AssignmentDurationInSeconds><NumberOfAssignmentsPending>0</NumberOfAssignmentsPending><NumberOfAssignmentsAvailable>1</NumberOfAssignmentsAvailable><NumberOfAssignmentsCompleted>0</NumberOfAssignmentsCompleted></HIT></SearchHITsResult></SearchHITsResponse>
`
