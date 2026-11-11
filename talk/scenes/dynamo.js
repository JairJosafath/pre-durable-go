var ROW=[['user#42','#file#','users/42/beach.jpg','the upload itself',1.6],['user#42','#index#beach#','users/42/beach.jpg','find it by: beach',3],['user#42','#index#sea#','users/42/beach.jpg','find it by: sea',3.6],['user#42','#label#','beach','a label this user has',3.3],['user#42','#label#','sea','a label this user has',3.9]];
var X1=76,X2=330,X3=1120,TY=176,RH=62,Q='GET /query?index=sea',Q2='begins_with(sk, "#index#sea")';
function draw(t){
var i,n=0,a0=P(t,.3,.8),br=eo(P(t,.6,1.2)),q=P(t,8.4,8.8);
g.font='500 24px '+MO;var cw=g.measureText('m').width;
for(i=0;5>i;i++){if(t>=ROW[i][4])n++}
txt('PrimeDrive catalog table',60,46,'600 30px '+SA,TX,a0);
var pu=1+.25*Math.sin(P(t,5.6,6.4)*3.1416);g.save();g.translate(1604,46);g.scale(pu,pu);txt('items: '+n,0,0,'600 28px '+MO,CY,a0,'right');g.restore();
txt('PRIMARY KEY',X1,TY-84,'600 24px '+SA,OR,P(t,.6,1));
if(br>0){g.strokeStyle=OR;g.lineWidth=3;g.beginPath();g.moveTo(X1,TY-54);g.lineTo(X1,TY-62);g.lineTo(X1+(X3-60-X1)*br,TY-62);if(br>=1)g.lineTo(X3-60,TY-54);g.stroke()}
txt('pk',X1,TY-28,'600 24px '+MO,MU,a0);txt('sk',X2,TY-28,'600 24px '+MO,MU,a0);txt('what it is',X3,TY-28,'600 24px '+SA,MU,a0);
seg(60,TY,1604,TY,LN,3,a0);
for(i=0;5>i;i++){var R=ROW[i],a=P(t,R[4],R[4]+.4),y=TY+14+i*RH,dx=(1-eo(a))*-40,hit=i===2,dim=q>0?(hit?1:1-.7*q):1,m=i>0?P(t,5.4+i*.2,5.6+i*.2)*(1-P(t,5.9+i*.2,6.5+i*.2)):0;
if(a>0){g.globalAlpha=a*dim;rr(60+dx,y,1544,52,12);g.fillStyle='#16222E';g.fill();g.strokeStyle=hit?(q>.5?CY:LN):(m>.2?OR:LN);if(m>.2)g.strokeStyle=OR;g.lineWidth=(hit?(q>.5?4:2):2)+(m>.2?2:0);g.stroke();g.globalAlpha=1;
txt(R[0],X1+dx,y+27,'500 24px '+MO,TX,a*dim);txt(R[1],X2+dx,y+27,'500 24px '+MO,OR,a*dim);txt(R[2],X2+dx+R[1].length*cw,y+27,'500 24px '+MO,TX,a*dim);txt(R[3],X3+dx,y+27,'500 24px '+SA,MU,a*dim)}
if(i>0){var k=P(t,5.2+i*.2,5.7+i*.2);if(k>0){if(1>k){g.globalAlpha=(1-k)*.9;rr(60,y-40*(1-eio(k)),1544,52,12);g.strokeStyle=OR;g.lineWidth=3;g.setLineDash([10,8]);g.stroke();g.setLineDash([]);g.globalAlpha=1}}}}
var qa=P(t,7.2,7.5);if(qa>0){g.globalAlpha=qa;rr(60,510,1544,56,12);g.strokeStyle=CY;g.lineWidth=2;g.stroke();g.globalAlpha=1;
txt(Q.slice(0,Math.floor(P(t,7.4,8.2)*Q.length)),X1,538,'600 24px '+MO,CY,1);txt(Q2,1588,538,'500 24px '+MO,TX,P(t,8.3,8.7),'right')}
var CAP=[[1.4,2.8,'An upload writes one item for the file.'],[2.8,5.2,'Each label adds an index item and a label item.'],[5.2,7.2,'A retried step writes the same keys: still five items.'],[7.2,99,'Search by label is a key-prefix query.']];
for(i=0;4>i;i++){txt(CAP[i][2],60,610,'500 30px '+SA,TX,win(t,CAP[i][0],CAP[i][1]))}
}
