var WH='#F6F4EE',D2R=.017453;
function st(a,b,k){var o=[],i;for(i=0;9>i;i++){o.push(mix(a[i],b[i],k))}return o}
function pos(s,phi,r){return [s[0]-r*Math.sin(phi*D2R),s[1]-r*Math.cos(phi*D2R)]}
function glyph(j,X,Y,sc,al,col){if(!(al>0))return;g.save();g.translate(X,Y);g.scale(sc,sc);g.globalAlpha=al;g.strokeStyle=col;g.fillStyle=col;g.lineWidth=4.5;g.lineCap='round';g.lineJoin='round';g.beginPath();
if(j===0){g.moveTo(-10,-13);g.lineTo(-22,0);g.lineTo(-10,13);g.moveTo(10,-13);g.lineTo(22,0);g.lineTo(10,13);g.moveTo(4,-17);g.lineTo(-4,17);g.stroke()}
if(j===1){g.moveTo(5,-21);g.lineTo(-11,3);g.lineTo(1,3);g.lineTo(-5,21);g.lineTo(11,-3);g.lineTo(-1,-3);g.closePath();g.fill()}
if(j===2){g.moveTo(0,-19);g.lineTo(21,17);g.lineTo(-21,17);g.closePath();g.stroke();g.beginPath();g.moveTo(0,-6);g.lineTo(0,5);g.stroke();g.beginPath();g.arc(0,11,1.5,0,6.2832);g.fill()}
if(j===3){g.moveTo(-17,17);g.lineTo(5,-5);g.stroke();g.beginPath();g.arc(11,-11,9,-.5,3.7);g.stroke()}
if(j===4){g.arc(0,-5,13,2.2,7.2);g.stroke();g.beginPath();g.moveTo(-6,13);g.lineTo(6,13);g.moveTo(-4,20);g.lineTo(4,20);g.stroke()}
g.restore()}
function wedge(s){var G=pos(s,s[6],s[2]+s[7]),q=g.createRadialGradient(G[0],G[1],0,G[0],G[1],s[8]);q.addColorStop(0,'rgba(0,173,216,.8)');q.addColorStop(.32,'rgba(255,122,154,.45)');q.addColorStop(.62,'rgba(242,166,90,.3)');q.addColorStop(1,'rgba(242,166,90,0)');g.fillStyle=q;g.beginPath();g.rect(0,0,BW,BH);g.arc(s[0],s[1],s[2],0,6.2832);g.fill('evenodd')}
function spot(X,Y,al){if(!(al>0))return;var q=g.createRadialGradient(X,Y,0,X,Y,62);q.addColorStop(0,'rgba(0,173,216,.9)');q.addColorStop(.6,'rgba(0,173,216,.85)');q.addColorStop(1,'rgba(0,173,216,0)');g.globalAlpha=al;g.fillStyle=q;g.beginPath();g.arc(X,Y,62,0,6.2832);g.fill();g.globalAlpha=1}
function draw(t){
var kA=eio(P(t,.45,1.5)),kC=eio(P(t,3,3.9)),s=st(st(S0,S1,kA),S2,kC),j,p,f;
g.fillStyle='#F6F4EE';g.beginPath();g.rect(0,0,BW,BH);g.arc(960,1100,1157,0,6.2832);g.fill('evenodd');
wedge(s);
for(j=0;5>j;j++){f=F0===F1?(j===F1?1:0):((j===F1?kA:0)+(j===F0?1-kA:0));p=pos(s,s[5]+j*s[4],s[3]);spot(p[0],p[1],f);glyph(j,p[0],p[1],mix(.8,1.3,f),mix(.6,1,f),f>.5?WH:'#006E8A')}
var tt=t-1.5,F=pos(s,s[5]+F1*s[4],s[3]),vis=1-P(t,2.9,3.3),dx=540-F[0],dy=466-F[1],d=Math.sqrt(dx*dx+dy*dy),ux=dx/d,uy=dy/d,kk=eo(P(tt,.1,.8)),r=P(tt,.6,1),pu=P(tt,.7,1.7);
seg(F[0]+ux*58,F[1]+uy*58,mix(F[0]+ux*58,540,kk),mix(F[1]+uy*58,466,kk),'#006E8A',4,(kk>0?1:0)*vis);
if(r*vis>0){g.save();g.translate(F[0],F[1]);g.rotate(t*.5);g.setLineDash([10,9]);g.globalAlpha=r*vis;g.beginPath();g.arc(0,0,52*bk(r),0,6.2832);g.strokeStyle='#006E8A';g.lineWidth=4;g.stroke();g.setLineDash([]);g.restore();g.globalAlpha=1}
if(pu>0){if(1>pu){ring(F[0],F[1],52+pu*30,'#006E8A',3,(1-pu)*vis)}}
}
