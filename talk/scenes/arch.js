var N=[['b',120,290,'browser'],['a',430,150,'API Gateway'],['s',430,430,'S3 uploads'],['l',720,64,'presigner'],['l',720,250,'reviewer'],['l',1020,430,'pipeline'],['d',1420,150,'DynamoDB'],['k',1420,318,'Rekognition'],['l',1420,486,'thumbnailer']];
var E=[[0,1,1],[1,3,1],[0,2,1],[2,5,2],[5,7,2],[5,8,2],[5,6,2],[0,1,3],[1,4,3],[4,5,3],[1,6,4]];
var PH=[[.9,3.7],[3.7,6.9],[6.9,9.9],[9.9,12.4]];
var CAP=['Upload: a presigned URL from the API, then straight to S3.','S3 starts the durable function. It analyses, thumbnails and records.','A review suspends it. The reviewer function sends the callback.','Search goes from API Gateway straight to DynamoDB.'];
function ends(e){var a=N[e[0]],b=N[e[1]],dx=b[1]-a[1],dy=b[2]-a[2],d=Math.sqrt(dx*dx+dy*dy),ux=dx/d,uy=dy/d,m=Math.max(Math.abs(ux),Math.abs(uy)),ra=((e[0]===5?52:40)+10)/m,rb=((e[1]===5?52:40)+16)/m;return [a[1]+ux*ra,a[2]+uy*ra,b[1]-ux*rb,b[2]-uy*rb,ux,uy]}
function node(n,al){if(!(al>0))return;var X=n[1],Y=n[2],big=n[3]==='pipeline',h=big?52:40;g.globalAlpha=al;rr(X-h,Y-h,2*h,2*h,14);g.fillStyle='#16222E';g.fill();g.strokeStyle=big?CY:LN;g.lineWidth=big?4:2;g.stroke();
if(n[0]==='b'){rr(X-24,Y-18,48,36,6);g.strokeStyle=TX;g.lineWidth=3;g.stroke();g.beginPath();g.moveTo(X-24,Y-7);g.lineTo(X+24,Y-7);g.stroke()}
else{g.save();g.translate(X-h*.8,Y-h*.8);g.scale(h*.02,h*.02);g.fillStyle=big?CY:TX;g.fill(IC[n[0]],'evenodd');g.restore()}
g.globalAlpha=1;txt(n[3],X,Y+h+24,'600 24px '+SA,big?CY:MU,al,'center');if(big)txt('durable function',X,Y+h+56,'500 24px '+SA,MU,al,'center')}
function draw(t){
var i,cnt=[0,0,0,0,0],a0=P(t,.2,.8),q;
for(i=0;E.length>i;i++){q=ends(E[i]);seg(q[0],q[1],q[2],q[3],LN,3,a0)}
for(i=0;E.length>i;i++){var e=E[i],ph=PH[e[2]-1],j=cnt[e[2]]++,k=eo(P(t,ph[0]+j*.6,ph[0]+j*.6+.8));if(k>0){q=ends(e);var X=mix(q[0],q[2],k),Y=mix(q[1],q[3],k),al=e[2]===4?1:1-.5*P(t,ph[1]-.3,ph[1]);seg(q[0],q[1],X,Y,CY,4,al);g.globalAlpha=al;g.beginPath();g.moveTo(X+q[4]*12,Y+q[5]*12);g.lineTo(X-q[4]*6-q[5]*9,Y-q[5]*6+q[4]*9);g.lineTo(X-q[4]*6+q[5]*9,Y-q[5]*6-q[4]*9);g.closePath();g.fillStyle=CY;g.fill();g.globalAlpha=1}}
for(i=0;N.length>i;i++){node(N[i],P(t,.15+i*.07,.55+i*.07))}
for(i=0;4>i;i++){txt(CAP[i],60,604,'500 30px '+SA,TX,win(t,PH[i][0],i===3?99:PH[i][1]))}
}
