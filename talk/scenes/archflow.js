var TE='#006E8A';
function plen(p){var i,L=0;for(i=1;p.length>i;i++){L+=Math.abs(p[i][0]-p[i-1][0])+Math.abs(p[i][1]-p[i-1][1])}return L}
function part(p,k){var i,L=plen(p)*k,x=p[0][0],y=p[0][1],ux=1,uy=0;g.beginPath();g.moveTo(x,y);
for(i=1;p.length>i;i++){var dx=p[i][0]-p[i-1][0],dy=p[i][1]-p[i-1][1],d=Math.abs(dx)+Math.abs(dy);if(!(d>0))continue;ux=dx/d;uy=dy/d;
if(L>=d){x=p[i][0];y=p[i][1];g.lineTo(x,y);L-=d}else{x=p[i-1][0]+ux*L;y=p[i-1][1]+uy*L;g.lineTo(x,y);break}}
g.stroke();return [x,y,ux,uy]}
function draw(t){
var i;g.lineJoin='round';g.lineCap='round';
for(i=0;HOT.length>i;i++){var h=HOT[i],a=P(t,.1+i*.12,.5+i*.12),s=h[2]/2+8;if(a>0){g.globalAlpha=a;rr(h[0]-s,h[1]-s,2*s,2*s,8);g.strokeStyle=TE;g.lineWidth=4;g.stroke();g.globalAlpha=1}}
for(i=0;EDG.length>i;i++){var k=eio(P(t,.3+i*.6,1.2+i*.6));if(k>0){g.strokeStyle=TE;g.fillStyle=TE;g.lineWidth=6;var q=part(EDG[i],k);
g.beginPath();g.moveTo(q[0]+q[2]*13,q[1]+q[3]*13);g.lineTo(q[0]-q[2]*5-q[3]*11,q[1]-q[3]*5+q[2]*11);g.lineTo(q[0]-q[2]*5+q[3]*11,q[1]-q[3]*5-q[2]*11);g.closePath();g.fill()}}
}
