var ROWS=[['Starting workflow',1],['Step 1 complete',2],['Resumed after wait',6.2]],EM=[1,2,4.2,5,6.2],GH=[[0,4.2],[1,5]];
function draw(t){
var i,rep=t>=3.6,pulse=0,pc=CY,n=0;
txt(rep?'invocation 2 · replay':'invocation 1',32,50,'600 32px '+SA,rep?OR:CY,rep?P(t,3.6,3.9):1-P(t,3.3,3.6));
for(i=0;5>i;i++){var p=P(t,EM[i],EM[i]+.15)*(1-P(t,EM[i]+.3,EM[i]+.7));if(p>pulse){pulse=p;pc=(i===2||i===3)?OR:CY}}
rr(32,92,696,64,14);g.fillStyle='#16222E';g.fill();g.strokeStyle=pulse>.2?pc:LN;g.lineWidth=pulse>.2?4:2;g.stroke();
txt('ctx.Logger().Info(...)',56,125,'500 24px '+MO,TX,1);
txt('LOG OUTPUT',32,214,'600 24px '+SA,MU,1);
for(i=0;3>i;i++){var R=ROWS[i],a=P(t,R[1],R[1]+.35),y=244+i*76;if(t>=R[1])n++;
if(a>0){g.globalAlpha=a;rr(32+(1-eo(a))*30,y,696,60,12);g.fillStyle='#16222E';g.fill();g.strokeStyle=LN;g.lineWidth=2;g.stroke();g.globalAlpha=1;txt(R[0],56+(1-eo(a))*30,y+31,'500 24px '+MO,TX,a)}}
for(i=0;2>i;i++){var G=GH[i],k=P(t,G[1],G[1]+.5),gy=mix(92,244+G[0]*76,eio(k));
if(k>0){if(1>k){g.globalAlpha=(1-k)*.9;rr(32,gy,696,60,12);g.strokeStyle=OR;g.lineWidth=3;g.setLineDash([8,8]);g.stroke();g.setLineDash([]);g.globalAlpha=1}}
txt('suppressed on replay',704,244+G[0]*76+31,'600 24px '+SA,OR,P(t,G[1]+.4,G[1]+.7),'right')}
txt('lines written: '+n,32,512,'600 30px '+MO,TX,P(t,1,1.3));
txt('Each line appears once, although the handler ran twice.',32,566,'500 24px '+SA,MU,P(t,6.6,7));
}
